package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/clinic/app"
	"github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
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
	// GET /v1/workspace - products catalog (Classic open, Ultimate closed)
	r.Get("/workspace", h.getWorkspace)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		if h.auth != nil {
			r.Use(h.auth.Require)
		}
		r.Get("/clinic/settings", h.getSettings)
		r.Patch("/clinic/settings", h.patchSettings)
	})
}

type productsResponse struct {
	Products []domain.Product `json:"products"`
}

type settingsResponse struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Locale      string `json:"locale"`
	Currency    string `json:"currency"`
	Palette     string `json:"palette"`
	LogoKey     string `json:"logo_key"`
	Phone       string `json:"phone"`
	MapURL      string `json:"map_url"`
	Intro       string `json:"intro"`
	Legal       string `json:"legal"`
}

func (h *Handler) getWorkspace(w http.ResponseWriter, r *http.Request) {
	prods := h.svc.GetProducts(r.Context())
	httpx.WriteJSON(w, http.StatusOK, productsResponse{Products: prods})
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	rawClinicID := ctxkey.ClinicID(r.Context())
	if rawClinicID == "" {
		httpx.WriteError(w, errx.ErrUnauthenticated)
		return
	}
	wsID, err := id.Parse(rawClinicID)
	if err != nil {
		httpx.WriteError(w, errx.ErrUnauthenticated)
		return
	}

	st, err := h.svc.GetSettings(r.Context(), wsID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toSettingsResponse(st))
}

type patchSettingsRequest struct {
	WorkspaceID *string `json:"workspace_id,omitempty"`
	Name        *string `json:"name"`
	Locale      *string `json:"locale"`
	Currency    *string `json:"currency"`
	Palette     *string `json:"palette"`
	LogoKey     *string `json:"logo_key"`
	Phone       *string `json:"phone"`
	MapURL      *string `json:"map_url"`
	Intro       *string `json:"intro"`
	Legal       *string `json:"legal"`
}

func (h *Handler) patchSettings(w http.ResponseWriter, r *http.Request) {
	rawClinicID := ctxkey.ClinicID(r.Context())
	if rawClinicID == "" {
		httpx.WriteError(w, errx.ErrUnauthenticated)
		return
	}
	wsID, err := id.Parse(rawClinicID)
	if err != nil {
		httpx.WriteError(w, errx.ErrUnauthenticated)
		return
	}

	var req patchSettingsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	// workspace_id cannot be taken from request body or changed to another workspace
	if req.WorkspaceID != nil && *req.WorkspaceID != "" && *req.WorkspaceID != rawClinicID {
		httpx.WriteError(w, errx.New("invalid_workspace_id", "workspace_id cannot be set from body", errx.ErrForbidden))
		return
	}

	out, err := h.svc.UpdateSettings(r.Context(), wsID, app.UpdateSettingsInput{
		Name:     req.Name,
		Locale:   req.Locale,
		Currency: req.Currency,
		Palette:  req.Palette,
		LogoKey:  req.LogoKey,
		Phone:    req.Phone,
		MapURL:   req.MapURL,
		Intro:    req.Intro,
		Legal:    req.Legal,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toSettingsResponse(out))
}

func toSettingsResponse(s domain.Settings) settingsResponse {
	return settingsResponse{
		WorkspaceID: s.WorkspaceID.String(),
		Name:        s.Name,
		Locale:      string(s.Locale),
		Currency:    string(s.Currency),
		Palette:     string(s.Palette),
		LogoKey:     s.LogoKey,
		Phone:       s.Phone,
		MapURL:      s.MapURL,
		Intro:       s.Intro,
		Legal:       s.Legal,
	}
}
