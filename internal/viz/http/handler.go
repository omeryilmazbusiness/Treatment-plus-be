package http

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/app"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/player"
)

type Handler struct {
	svc  *app.Service
	auth *authn.JWT
}

func NewHandler(svc *app.Service, auth *authn.JWT) *Handler {
	return &Handler{svc: svc, auth: auth}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/viz/recipes", h.recipes)
	r.Get("/viz/demo", h.demo)
	r.Get("/viz/player", h.playerHTML)
	r.Get("/viz/player.js", h.playerJS)
	r.Get("/viz/cinematics.js", h.cinematicsJS)
	r.Get("/viz/mouth.glb", h.mouth)

	r.Group(func(r chi.Router) {
		if h.auth != nil {
			r.Use(h.auth.Require)
		}
		r.Get("/plans/{id}/viz", h.playlist)
	})
}

func (h *Handler) recipes(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"recipes": h.svc.Recipes()})
}

func (h *Handler) demo(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pl, err := h.svc.Demo(q.Get("code"), q.Get("locale"), q.Get("teeth"), q.Get("items"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pl)
}

func servePlayerFile(w http.ResponseWriter, name, contentType string, embedded []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	disk := filepath.Join("internal", "viz", "player", "static", name)
	if b, err := os.ReadFile(disk); err == nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(embedded)
}

func (h *Handler) playerHTML(w http.ResponseWriter, _ *http.Request) {
	servePlayerFile(w, "index.html", "text/html; charset=utf-8", player.HTML)
}

func (h *Handler) playerJS(w http.ResponseWriter, _ *http.Request) {
	servePlayerFile(w, "player.js", "text/javascript; charset=utf-8", player.JS)
}

func (h *Handler) cinematicsJS(w http.ResponseWriter, _ *http.Request) {
	servePlayerFile(w, "cinematics.js", "text/javascript; charset=utf-8", player.Cinematics)
}

func (h *Handler) mouth(w http.ResponseWriter, r *http.Request) {
	paths := []string{
		os.Getenv("VIZ_MODEL"),
		filepath.Join("internal", "viz", "player", "static", "mouth.glb"),
		filepath.Join(os.Getenv("HOME"), "Desktop", "mouth.glb"),
	}
	for _, p := range paths {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			w.Header().Set("Content-Type", "model/gltf-binary")
			w.Header().Set("Cache-Control", "no-store")
			http.ServeFile(w, r, p)
			return
		}
	}
	httpx.WriteError(w, errx.New("model_missing", "mouth.glb not exported from the blend yet", errx.ErrNotFound))
}

func (h *Handler) playlist(w http.ResponseWriter, r *http.Request) {
	raw := ctxkey.ClinicID(r.Context())
	if raw == "" {
		httpx.WriteError(w, errx.ErrUnauthenticated)
		return
	}
	wsID, err := id.Parse(raw)
	if err != nil {
		httpx.WriteError(w, errx.ErrUnauthenticated)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "plan id required", errx.ErrInvalid))
		return
	}
	pl, err := h.svc.Playlist(r.Context(), wsID, planID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pl)
}
