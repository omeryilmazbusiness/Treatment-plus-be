package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/output/app"
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
		r.Post("/plans/{id}/pdf", h.generate)
		r.Get("/plans/{id}/pdf", h.download)
	})
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	wsID, planID, err := ids(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	doc, err := h.svc.Generate(r.Context(), wsID, planID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"plan_id":      doc.PlanID.String(),
		"content_type": doc.ContentType,
		"page_count":   doc.PageCount,
		"size":         doc.Size(),
	})
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	wsID, planID, err := ids(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	doc, err := h.svc.Generate(r.Context(), wsID, planID)
	if err != nil {
		doc, err = h.svc.Get(r.Context(), wsID, planID)
	}
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", doc.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="treatment-plan.pdf"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(doc.Bytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc.Bytes)
}

func ids(r *http.Request) (id.ID, id.ID, error) {
	raw := ctxkey.ClinicID(r.Context())
	if raw == "" {
		return id.ID{}, id.ID{}, errx.ErrUnauthenticated
	}
	wsID, err := id.Parse(raw)
	if err != nil {
		return id.ID{}, id.ID{}, errx.ErrUnauthenticated
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id.ID{}, id.ID{}, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid)
	}
	return wsID, planID, nil
}
