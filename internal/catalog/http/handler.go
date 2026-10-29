package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/catalog/app"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
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
		// Diagnoses
		r.Get("/catalog/diagnoses", h.listDiagnoses)
		r.Post("/catalog/diagnoses", h.createDiagnosis)
		r.Patch("/catalog/diagnoses/{id}", h.patchDiagnosis)

		// Treatments
		r.Get("/catalog/treatments", h.listTreatments)
		r.Post("/catalog/treatments", h.createTreatment)
		r.Patch("/catalog/treatments/{id}", h.patchTreatment)
		r.Delete("/catalog/treatments/{id}", h.deleteTreatment)

		// Price lists
		r.Get("/catalog/price-lists", h.listPriceLists)
		r.Post("/catalog/price-lists", h.createPriceList)
		r.Get("/catalog/price-lists/{id}", h.getPriceList)
		r.Put("/catalog/price-lists/{id}/items", h.setPriceItems)
	})
}

func (h *Handler) getWorkspaceID(r *http.Request) (id.ID, error) {
	raw := ctxkey.ClinicID(r.Context())
	if raw == "" {
		return id.ID{}, errx.ErrUnauthenticated
	}
	return id.Parse(raw)
}

func (h *Handler) listDiagnoses(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	inc := r.URL.Query().Get("include_inactive") == "true"
	list, err := h.svc.ListDiagnoses(r.Context(), wsID, inc)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"diagnoses": list})
}

type createDiagnosisRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (h *Handler) createDiagnosis(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req createDiagnosisRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	diag, err := h.svc.CreateDiagnosis(r.Context(), wsID, req.Code, req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, diag)
}

type patchDiagnosisRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

func (h *Handler) patchDiagnosis(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	diagID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed diagnosis id", errx.ErrInvalid))
		return
	}
	var req patchDiagnosisRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	diag, err := h.svc.UpdateDiagnosis(r.Context(), wsID, diagID, req.Name, req.IsActive)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, diag)
}

func (h *Handler) listTreatments(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	inc := r.URL.Query().Get("include_inactive") == "true"
	list, err := h.svc.ListTreatments(r.Context(), wsID, inc)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"treatments": list})
}

type createTreatmentRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (h *Handler) createTreatment(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req createTreatmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.CreateTreatment(r.Context(), wsID, req.Code, req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

type patchTreatmentRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

func (h *Handler) patchTreatment(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	treatID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed treatment id", errx.ErrInvalid))
		return
	}
	var req patchTreatmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.UpdateTreatment(r.Context(), wsID, treatID, req.Name, req.IsActive)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTreatment(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	treatID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed treatment id", errx.ErrInvalid))
		return
	}
	if err := h.svc.DeleteTreatment(r.Context(), wsID, treatID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) listPriceLists(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	list, err := h.svc.ListPriceLists(r.Context(), wsID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"price_lists": list})
}

type createPriceListRequest struct {
	Name      string `json:"name"`
	Currency  string `json:"currency"`
	IsDefault bool   `json:"is_default"`
}

func (h *Handler) createPriceList(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req createPriceListRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	curr := money.Currency(req.Currency)
	if !money.ValidCurrency(curr) {
		curr = money.TRY
	}
	pl, err := h.svc.CreatePriceList(r.Context(), wsID, req.Name, curr, req.IsDefault)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, pl)
}

func (h *Handler) getPriceList(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	listID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed price list id", errx.ErrInvalid))
		return
	}
	pl, items, err := h.svc.GetPriceList(r.Context(), wsID, listID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"price_list": pl,
		"items":      items,
	})
}

type setPriceItemRequest struct {
	TreatmentID string `json:"treatment_id"`
	Amount      int64  `json:"amount"` // minor unit
}

func (h *Handler) setPriceItems(w http.ResponseWriter, r *http.Request) {
	wsID, err := h.getWorkspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	listID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed price list id", errx.ErrInvalid))
		return
	}
	var req []setPriceItemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	var inputs []app.SetPriceItemInput
	for _, it := range req {
		trID, err := id.Parse(it.TreatmentID)
		if err != nil {
			httpx.WriteError(w, errx.New("invalid_treatment_id", "invalid treatment id", errx.ErrInvalid))
			return
		}
		inputs = append(inputs, app.SetPriceItemInput{
			TreatmentID: trID,
			Amount:      it.Amount,
		})
	}
	items, err := h.svc.SetPriceItems(r.Context(), wsID, listID, inputs)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
