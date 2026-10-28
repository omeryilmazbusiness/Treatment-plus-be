package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
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
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.register)
		r.Post("/login", h.login)
		r.Post("/refresh", h.refresh)
		r.Post("/logout", h.logout)
		r.Group(func(r chi.Router) {
			r.Use(h.auth.Require)
			r.Get("/me", h.me)
		})
	})
	r.Group(func(r chi.Router) {
		r.Use(h.auth.Require)
		r.Get("/workspaces/{workspaceID}", h.getWorkspace)
		r.Get("/clinics/{clinicID}", h.getClinic)
	})
}

type clinicStatRequest struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type clinicDoctorRequest struct {
	Name     string              `json:"name"`
	Title    string              `json:"title"`
	Subtitle string              `json:"subtitle"`
	Bio      string              `json:"bio"`
	PhotoKey string              `json:"photo_key"`
	Stats    []clinicStatRequest `json:"stats"`
}

type clinicLabRequest struct {
	Title    string              `json:"title"`
	Text     string              `json:"text"`
	ImageKey string              `json:"image_key"`
	Stats    []clinicStatRequest `json:"stats"`
}

type clinicAwardsRequest struct {
	Text      string   `json:"text"`
	BadgeKeys []string `json:"badge_keys"`
}

type clinicHotelRequest struct {
	Name     string `json:"name"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key"`
}

type clinicProfileRequest struct {
	Slogan      string              `json:"slogan"`
	Phone       string              `json:"phone"`
	Email       string              `json:"email"`
	Website     string              `json:"website"`
	Address     string              `json:"address"`
	Hours       string              `json:"hours"`
	Locale      string              `json:"locale"`
	Currency    string              `json:"currency"`
	Palette     string              `json:"palette"`
	LogoKey     string              `json:"logo_key"`
	CoverKey    string              `json:"cover_key"`
	MapURL      string              `json:"map_url"`
	Intro       string              `json:"intro"`
	Legal       string              `json:"legal"`
	AboutText   string              `json:"about_text"`
	AboutKey    string              `json:"about_key"`
	WelcomeBody string              `json:"welcome_body"`
	SignerName  string              `json:"signer_name"`
	SignerTitle string              `json:"signer_title"`
	Highlights  []clinicStatRequest `json:"highlights"`
	Doctor      clinicDoctorRequest `json:"doctor"`
	Lab         clinicLabRequest    `json:"lab"`
	Awards      clinicAwardsRequest `json:"awards"`
	Hotel       clinicHotelRequest  `json:"hotel"`
}

func toClinicStats(in []clinicStatRequest) []app.ClinicStat {
	var out []app.ClinicStat
	for _, s := range in {
		out = append(out, app.ClinicStat{Value: s.Value, Label: s.Label})
	}
	return out
}

func (c clinicProfileRequest) toProfile() app.ClinicProfile {
	return app.ClinicProfile{
		Slogan:      c.Slogan,
		Phone:       c.Phone,
		Email:       c.Email,
		Website:     c.Website,
		Address:     c.Address,
		Hours:       c.Hours,
		Locale:      c.Locale,
		Currency:    c.Currency,
		Palette:     c.Palette,
		LogoKey:     c.LogoKey,
		CoverKey:    c.CoverKey,
		MapURL:      c.MapURL,
		Intro:       c.Intro,
		Legal:       c.Legal,
		AboutText:   c.AboutText,
		AboutKey:    c.AboutKey,
		WelcomeBody: c.WelcomeBody,
		SignerName:  c.SignerName,
		SignerTitle: c.SignerTitle,
		Highlights:  toClinicStats(c.Highlights),
		Doctor: app.ClinicDoctor{
			Name: c.Doctor.Name, Title: c.Doctor.Title, Subtitle: c.Doctor.Subtitle,
			Bio: c.Doctor.Bio, PhotoKey: c.Doctor.PhotoKey, Stats: toClinicStats(c.Doctor.Stats),
		},
		Lab: app.ClinicLab{
			Title: c.Lab.Title, Text: c.Lab.Text, ImageKey: c.Lab.ImageKey, Stats: toClinicStats(c.Lab.Stats),
		},
		Awards: app.ClinicAwards{Text: c.Awards.Text, BadgeKeys: c.Awards.BadgeKeys},
		Hotel:  app.ClinicHotel{Name: c.Hotel.Name, Text: c.Hotel.Text, ImageKey: c.Hotel.ImageKey},
	}
}

type registerRequest struct {
	Name     string               `json:"name"`
	Email    string               `json:"email"`
	Password string               `json:"password"`
	Role     string               `json:"role"`
	Clinic   clinicProfileRequest `json:"clinic"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type userBody struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type workspaceBody struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type tokensBody struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type authResponse struct {
	User      userBody      `json:"user"`
	Workspace workspaceBody `json:"workspace"`
	Tokens    tokensBody    `json:"tokens"`
}

type meResponse struct {
	User      userBody      `json:"user"`
	Workspace workspaceBody `json:"workspace"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := h.svc.Register(r.Context(), app.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		Clinic:   req.Clinic.toProfile(),
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toAuth(out))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := h.svc.Login(r.Context(), app.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAuth(out))
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAuth(out))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Me(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toMe(out))
}

func (h *Handler) getClinic(w http.ResponseWriter, r *http.Request) {
	h.writeWorkspace(w, r, chi.URLParam(r, "clinicID"))
}

func (h *Handler) getWorkspace(w http.ResponseWriter, r *http.Request) {
	h.writeWorkspace(w, r, chi.URLParam(r, "workspaceID"))
}

func (h *Handler) writeWorkspace(w http.ResponseWriter, r *http.Request, rawID string) {
	requested, err := id.Parse(rawID)
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "workspace id is invalid", errx.ErrInvalid))
		return
	}
	out, err := h.svc.GetWorkspace(r.Context(), requested)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toMe(out))
}

func toAuth(out app.AuthResult) authResponse {
	return authResponse{
		User:      userJSON(out.Account),
		Workspace: workspaceJSON(out.Account),
		Tokens: tokensBody{
			AccessToken:  out.Tokens.Access,
			RefreshToken: out.Tokens.Refresh,
			TokenType:    "Bearer",
			ExpiresIn:    out.Tokens.ExpiresIn,
		},
	}
}

func toMe(out app.AccountView) meResponse {
	return meResponse{User: userJSON(out), Workspace: workspaceJSON(out)}
}

func userJSON(a app.AccountView) userBody {
	return userBody{ID: a.User.ID.String(), Email: a.User.Email, Role: string(a.User.Role)}
}

func workspaceJSON(a app.AccountView) workspaceBody {
	return workspaceBody{ID: a.Workspace.ID.String(), Name: a.Workspace.Name, Kind: string(a.Workspace.Kind)}
}
