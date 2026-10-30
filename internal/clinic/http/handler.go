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
	if h.auth != nil {
		r.With(h.auth.Optional).Get("/workspace", h.getWorkspace)
	} else {
		r.Get("/workspace", h.getWorkspace)
	}

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
	WorkspaceID string                   `json:"workspace_id"`
	Name        string                   `json:"name"`
	Slogan      string                   `json:"slogan"`
	Locale      string                   `json:"locale"`
	Currency    string                   `json:"currency"`
	Palette     string                   `json:"palette"`
	LogoKey     string                   `json:"logo_key"`
	CoverKey    string                   `json:"cover_key"`
	Phone       string                   `json:"phone"`
	Email       string                   `json:"email"`
	Website     string                   `json:"website"`
	Address     string                   `json:"address"`
	Hours       string                   `json:"hours"`
	MapURL      string                   `json:"map_url"`
	Intro       string                   `json:"intro"`
	Legal       string                   `json:"legal"`
	AboutText   string                   `json:"about_text"`
	AboutKey    string                   `json:"about_key"`
	WelcomeBody string                   `json:"welcome_body"`
	SignerName  string                   `json:"signer_name"`
	SignerTitle string                   `json:"signer_title"`
	Highlights  []domain.Stat            `json:"highlights"`
	Doctor      domain.Doctor            `json:"doctor"`
	Lab         domain.Lab               `json:"lab"`
	Awards      domain.Awards            `json:"awards"`
	Hotel       domain.Hotel             `json:"hotel"`
	Team        []domain.Doctor          `json:"team"`
	Reviews     []domain.Review          `json:"reviews"`
	BeforeAfter []domain.BeforeAfterCase `json:"before_after"`
	CTA         domain.CTAConfig         `json:"cta"`
}

type patchSettingsRequest struct {
	WorkspaceID *string                   `json:"workspace_id,omitempty"`
	Name        *string                   `json:"name"`
	Slogan      *string                   `json:"slogan"`
	Locale      *string                   `json:"locale"`
	Currency    *string                   `json:"currency"`
	Palette     *string                   `json:"palette"`
	LogoKey     *string                   `json:"logo_key"`
	CoverKey    *string                   `json:"cover_key"`
	Phone       *string                   `json:"phone"`
	Email       *string                   `json:"email"`
	Website     *string                   `json:"website"`
	Address     *string                   `json:"address"`
	Hours       *string                   `json:"hours"`
	MapURL      *string                   `json:"map_url"`
	Intro       *string                   `json:"intro"`
	Legal       *string                   `json:"legal"`
	AboutText   *string                   `json:"about_text"`
	AboutKey    *string                   `json:"about_key"`
	WelcomeBody *string                   `json:"welcome_body"`
	SignerName  *string                   `json:"signer_name"`
	SignerTitle *string                   `json:"signer_title"`
	Highlights  *[]domain.Stat            `json:"highlights"`
	Doctor      *domain.Doctor            `json:"doctor"`
	Lab         *domain.Lab               `json:"lab"`
	Awards      *domain.Awards            `json:"awards"`
	Hotel       *domain.Hotel             `json:"hotel"`
	Team        *[]domain.Doctor          `json:"team"`
	Reviews     *[]domain.Review          `json:"reviews"`
	BeforeAfter *[]domain.BeforeAfterCase `json:"before_after"`
	CTA         *domain.CTAConfig         `json:"cta"`
}

func (h *Handler) getWorkspace(w http.ResponseWriter, r *http.Request) {
	rawClinicID := ctxkey.ClinicID(r.Context())
	var wsID id.ID
	if rawClinicID != "" {
		wsID, _ = id.Parse(rawClinicID)
	}
	prods := h.svc.GetProductsForWorkspace(r.Context(), wsID)
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

	if req.WorkspaceID != nil && *req.WorkspaceID != "" && *req.WorkspaceID != rawClinicID {
		httpx.WriteError(w, errx.New("invalid_workspace_id", "workspace_id cannot be set from body", errx.ErrForbidden))
		return
	}

	out, err := h.svc.UpdateSettings(r.Context(), wsID, app.UpdateSettingsInput{
		Name:        req.Name,
		Slogan:      req.Slogan,
		Locale:      req.Locale,
		Currency:    req.Currency,
		Palette:     req.Palette,
		LogoKey:     req.LogoKey,
		CoverKey:    req.CoverKey,
		Phone:       req.Phone,
		Email:       req.Email,
		Website:     req.Website,
		Address:     req.Address,
		Hours:       req.Hours,
		MapURL:      req.MapURL,
		Intro:       req.Intro,
		Legal:       req.Legal,
		AboutText:   req.AboutText,
		AboutKey:    req.AboutKey,
		WelcomeBody: req.WelcomeBody,
		SignerName:  req.SignerName,
		SignerTitle: req.SignerTitle,
		Highlights:  req.Highlights,
		Doctor:      req.Doctor,
		Lab:         req.Lab,
		Awards:      req.Awards,
		Hotel:       req.Hotel,
		Team:        req.Team,
		Reviews:     req.Reviews,
		BeforeAfter: req.BeforeAfter,
		CTA:         req.CTA,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toSettingsResponse(out))
}

func toSettingsResponse(s domain.Settings) settingsResponse {
	highlights := s.Highlights
	if highlights == nil {
		highlights = []domain.Stat{}
	}
	team := s.Team
	if team == nil {
		team = []domain.Doctor{}
	}
	reviews := s.Reviews
	if reviews == nil {
		reviews = []domain.Review{}
	}
	beforeAfter := s.BeforeAfter
	if beforeAfter == nil {
		beforeAfter = []domain.BeforeAfterCase{}
	}
	return settingsResponse{
		WorkspaceID: s.WorkspaceID.String(),
		Name:        s.Name,
		Slogan:      s.Slogan,
		Locale:      string(s.Locale),
		Currency:    string(s.Currency),
		Palette:     string(s.Palette),
		LogoKey:     s.LogoKey,
		CoverKey:    s.CoverKey,
		Phone:       s.Phone,
		Email:       s.Email,
		Website:     s.Website,
		Address:     s.Address,
		Hours:       s.Hours,
		MapURL:      s.MapURL,
		Intro:       s.Intro,
		Legal:       s.Legal,
		AboutText:   s.AboutText,
		AboutKey:    s.AboutKey,
		WelcomeBody: s.WelcomeBody,
		SignerName:  s.SignerName,
		SignerTitle: s.SignerTitle,
		Highlights:  highlights,
		Doctor:      s.Doctor,
		Lab:         s.Lab,
		Awards:      s.Awards,
		Hotel:       s.Hotel,
		Team:        team,
		Reviews:     reviews,
		BeforeAfter: beforeAfter,
		CTA:         s.CTA,
	}
}
