package http

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/media/app"
	"github.com/treatment-plus/treatment-plus-be/internal/media/domain"
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
		r.Post("/media", h.upload)
	})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	wsID, err := workspaceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := r.ParseMultipartForm(domain.MaxBytes + 1<<20); err != nil {
		httpx.WriteError(w, errx.New("invalid_media", "multipart image is required", errx.ErrInvalid))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_media", "file field is required", errx.ErrInvalid))
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, domain.MaxBytes+1))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	kind := r.FormValue("kind")
	ct := header.Header.Get("Content-Type")
	asset, err := h.svc.Upload(r.Context(), wsID, kind, ct, raw)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"key":          asset.Key,
		"kind":         asset.Kind,
		"content_type": asset.ContentType,
		"size":         len(asset.Bytes),
	})
}

func workspaceID(r *http.Request) (id.ID, error) {
	raw := ctxkey.ClinicID(r.Context())
	if raw == "" {
		return id.ID{}, errx.ErrUnauthenticated
	}
	return id.Parse(raw)
}
