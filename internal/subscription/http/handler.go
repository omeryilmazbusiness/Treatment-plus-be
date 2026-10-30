package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/app"
)

type Handler struct {
	svc  *app.Service
	auth *authn.JWT
}

func NewHandler(svc *app.Service, auth *authn.JWT) *Handler {
	return &Handler{svc: svc, auth: auth}
}

func (h *Handler) Register(r chi.Router) {
	// Public routes
	r.Get("/public/subscriptions/plans", h.listPlans)
	r.Post("/public/demo-requests", h.submitDemoRequest)

	// Authenticated clinic subscription management
	r.Group(func(r chi.Router) {
		if h.auth != nil {
			r.Use(h.auth.Require)
		}
		r.Get("/subscriptions/plans", h.listPlans)
		r.Get("/subscriptions/current", h.getCurrentSubscription)
		r.Post("/subscriptions/subscribe", h.subscribe)
		r.Post("/subscriptions/change-tier", h.changeTier)
		r.Post("/subscriptions/cancel", h.cancelSubscription)
	})
}

func (h *Handler) listPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.ListPlans(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"plans": plans})
}

func (h *Handler) getCurrentSubscription(w http.ResponseWriter, r *http.Request) {
	clinicID, err := h.getClinicID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	dto, err := h.svc.GetCurrentSubscription(r.Context(), clinicID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dto)
}

type subscribeRequest struct {
	PlanID string `json:"plan_id"`
}

func (h *Handler) subscribe(w http.ResponseWriter, r *http.Request) {
	clinicID, err := h.getClinicID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req subscribeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	dto, err := h.svc.Subscribe(r.Context(), clinicID, req.PlanID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dto)
}

func (h *Handler) changeTier(w http.ResponseWriter, r *http.Request) {
	clinicID, err := h.getClinicID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req subscribeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	dto, err := h.svc.ChangeTier(r.Context(), clinicID, req.PlanID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dto)
}

func (h *Handler) cancelSubscription(w http.ResponseWriter, r *http.Request) {
	clinicID, err := h.getClinicID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	dto, err := h.svc.CancelSubscription(r.Context(), clinicID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dto)
}

type submitDemoRequest struct {
	WebsiteURL  string `json:"website_url"`
	ClinicName  string `json:"clinic_name"`
	ContactName string `json:"contact_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Notes       string `json:"notes"`
}

func (h *Handler) submitDemoRequest(w http.ResponseWriter, r *http.Request) {
	var req submitDemoRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	demo, err := h.svc.SubmitDemoRequest(r.Context(), app.SubmitDemoInput{
		WebsiteURL:  req.WebsiteURL,
		ClinicName:  req.ClinicName,
		ContactName: req.ContactName,
		Email:       req.Email,
		Phone:       req.Phone,
		Notes:       req.Notes,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, demo)
}

func (h *Handler) getClinicID(r *http.Request) (id.ID, error) {
	raw := ctxkey.ClinicID(r.Context())
	if raw == "" {
		return id.ID{}, errx.ErrUnauthenticated
	}
	parsed, err := id.Parse(raw)
	if err != nil {
		return id.ID{}, errx.ErrUnauthenticated
	}
	return parsed, nil
}

