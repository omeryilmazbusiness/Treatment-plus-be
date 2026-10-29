package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/app"
	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Handler struct {
	svc  *app.Service
	auth *authn.JWT
}

func NewHandler(svc *app.Service, auth *authn.JWT) *Handler {
	return &Handler{svc: svc, auth: auth}
}

func (h *Handler) Register(r chi.Router) {
	r.Group(func(r chi.Router) {
		if h.auth != nil {
			r.Use(h.auth.Require)
		}
		// Patients
		r.Get("/patients", h.listPatients)
		r.Post("/patients", h.createPatient)
		r.Get("/patients/{id}", h.getPatient)
		r.Patch("/patients/{id}", h.patchPatient)

		// Classic form — static clinic brand is read-only; only plan fields are collected.
		r.Get("/plans/classic/form", h.classicForm)
		r.Post("/plans/classic", h.createClassic)

		// Ultimate form — includes welcome letter, video, before/after cases and 3D scenes.
		r.Get("/plans/ultimate/form", h.ultimateForm)
		r.Post("/plans/ultimate", h.createUltimate)

		// Plans
		r.Get("/plans", h.listPlans)
		r.Post("/plans", h.createPlan)
		r.Get("/plans/{id}", h.getPlan)
		r.Patch("/plans/{id}", h.patchPlan)
		r.Put("/plans/{id}/visits", h.putVisits)
		r.Put("/plans/{id}/items", h.putItems)
		r.Put("/plans/{id}/discounts", h.putDiscounts)
		r.Post("/plans/{id}/ready", h.setReady)
	})
}

func (h *Handler) getWorkspaceID(r *http.Request) (id.ID, error) {
	raw := ctxkey.ClinicID(r.Context())
	if raw == "" {
		return id.ID{}, errx.ErrUnauthenticated
	}
	return id.Parse(raw)
}

// Patient handlers
func (h *Handler) listPatients(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	list, err := h.svc.ListPatients(r.Context(), wsID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"patients": list})
}

type createPatientRequest struct {
	Name    string `json:"name"`
	Contact string `json:"contact"`
}

func (h *Handler) createPatient(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req createPatientRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.CreatePatient(r.Context(), wsID, req.Name, req.Contact)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) getPatient(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	patID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed patient id", errx.ErrInvalid))
		return
	}
	p, err := h.svc.GetPatient(r.Context(), wsID, patID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

type patchPatientRequest struct {
	Name    *string `json:"name"`
	Contact *string `json:"contact"`
}

func (h *Handler) patchPatient(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	patID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed patient id", errx.ErrInvalid))
		return
	}
	var req patchPatientRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.UpdatePatient(r.Context(), wsID, patID, req.Name, req.Contact)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

// Plan handlers
func (h *Handler) listPlans(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	list, err := h.svc.ListPlans(r.Context(), wsID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"plans": list})
}

func (h *Handler) classicForm(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	form, err := h.svc.ClassicForm(r.Context(), wsID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, form)
}

type classicPlanRequest struct {
	Palette   string                 `json:"palette"`
	Locale    string                 `json:"locale"`
	Patient   createPatientRequest   `json:"patient"`
	Title     string                 `json:"title"`
	Diagnoses []string               `json:"diagnoses"`
	Situation domain.Situation       `json:"situation"`
	Visits    []app.PutVisitInput    `json:"visits"`
	Items     []app.PutItemInput     `json:"items"`
	Discounts []app.PutDiscountInput `json:"discounts"`
}

func (h *Handler) createClassic(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req classicPlanRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	pl, err := h.svc.CreateClassic(r.Context(), wsID, app.ClassicPlanInput{
		Palette:   req.Palette,
		Locale:    req.Locale,
		Patient:   app.ClassicPatientInput{Name: req.Patient.Name, Contact: req.Patient.Contact},
		Title:     req.Title,
		Diagnoses: req.Diagnoses,
		Situation: req.Situation,
		Visits:    req.Visits,
		Items:     req.Items,
		Discounts: req.Discounts,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, pl)
}

func (h *Handler) ultimateForm(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	form, err := h.svc.UltimateForm(r.Context(), wsID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, form)
}

type ultimatePlanRequest struct {
	Palette            string                 `json:"palette"`
	Locale             string                 `json:"locale"`
	Patient            createPatientRequest   `json:"patient"`
	Title              string                 `json:"title"`
	WelcomeLetter      string                 `json:"welcome_letter"`
	VideoURL           string                 `json:"video_url"`
	BeforeAfterCaseIDs []string               `json:"before_after_case_ids"`
	Diagnoses          []string               `json:"diagnoses"`
	Situation          domain.Situation       `json:"situation"`
	Visits             []app.PutVisitInput    `json:"visits"`
	Items              []app.PutItemInput     `json:"items"`
	Discounts          []app.PutDiscountInput `json:"discounts"`
}

func (h *Handler) createUltimate(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req ultimatePlanRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	pl, err := h.svc.CreateUltimate(r.Context(), wsID, app.UltimatePlanInput{
		Palette:            req.Palette,
		Locale:             req.Locale,
		Patient:            app.ClassicPatientInput{Name: req.Patient.Name, Contact: req.Patient.Contact},
		Title:              req.Title,
		WelcomeLetter:      req.WelcomeLetter,
		VideoURL:           req.VideoURL,
		BeforeAfterCaseIDs: req.BeforeAfterCaseIDs,
		Diagnoses:          req.Diagnoses,
		Situation:          req.Situation,
		Visits:             req.Visits,
		Items:              req.Items,
		Discounts:          req.Discounts,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, pl)
}

type createPlanRequest struct {
	PatientID string           `json:"patient_id"`
	Title     string           `json:"title"`
	Currency  string           `json:"currency"`
	Palette   string           `json:"palette"`
	Locale    string           `json:"locale"`
	Diagnoses []string         `json:"diagnoses"`
	Situation domain.Situation `json:"situation"`
}

func (h *Handler) createPlan(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req createPlanRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	patID, err := id.Parse(req.PatientID)
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_patient_id", "invalid patient id", errx.ErrInvalid))
		return
	}
	pl, err := h.svc.CreatePlan(r.Context(), wsID, app.CreatePlanInput{
		PatientID: patID,
		Title:     req.Title,
		Currency:  req.Currency,
		Palette:   req.Palette,
		Locale:    req.Locale,
		Diagnoses: req.Diagnoses,
		Situation: req.Situation,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, pl)
}

func (h *Handler) getPlan(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}
	pl, err := h.svc.GetPlan(r.Context(), wsID, planID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pl)
}

type patchPlanRequest struct {
	Title              *string           `json:"title"`
	Diagnoses          *[]string         `json:"diagnoses"`
	Palette            *string           `json:"palette"`
	Locale             *string           `json:"locale"`
	Currency           *string           `json:"currency"`
	WelcomeLetter      *string           `json:"welcome_letter"`
	VideoURL           *string           `json:"video_url"`
	BeforeAfterCaseIDs *[]string         `json:"before_after_case_ids"`
	Situation          *domain.Situation `json:"situation"`
}

func (h *Handler) patchPlan(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}
	var req patchPlanRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	pl, err := h.svc.UpdatePlanHeader(r.Context(), wsID, planID, app.UpdatePlanHeaderInput{
		Title:              req.Title,
		Diagnoses:          req.Diagnoses,
		Palette:            req.Palette,
		Locale:             req.Locale,
		Currency:           req.Currency,
		WelcomeLetter:      req.WelcomeLetter,
		VideoURL:           req.VideoURL,
		BeforeAfterCaseIDs: req.BeforeAfterCaseIDs,
		Situation:          req.Situation,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pl)
}

type putVisitsRequest struct {
	Visits []app.PutVisitInput `json:"visits"`
}

func (h *Handler) putVisits(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}
	var req putVisitsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	pl, err := h.svc.PutVisits(r.Context(), wsID, planID, req.Visits)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pl)
}

type putItemsRequest struct {
	Items []app.PutItemInput `json:"items"`
}

func (h *Handler) putItems(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}
	var req putItemsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	pl, err := h.svc.PutItems(r.Context(), wsID, planID, req.Items)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pl)
}

type putDiscountsRequest struct {
	Discounts []app.PutDiscountInput `json:"discounts"`
}

func (h *Handler) putDiscounts(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}
	var req putDiscountsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	pl, err := h.svc.PutDiscounts(r.Context(), wsID, planID, req.Discounts)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pl)
}

func (h *Handler) setReady(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}
	pl, err := h.svc.SetReady(r.Context(), wsID, planID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pl)
}
