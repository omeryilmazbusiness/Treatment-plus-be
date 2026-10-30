package http

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/app"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/player"
)

// Public GitHub Release asset. The 142MB GLB exceeds GitHub's 100MB git limit,
// so it is not committed; collaborators download it from this URL.
const defaultMouthURL = "https://github.com/omeryilmazbusiness/Treatment-plus-be/releases/download/mouth-glb/mouth.glb"

var (
	mouthOnce sync.Once
	mouthPath string
	mouthErr  error
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
	p, err := ensureMouthModel()
	if err != nil || p == "" {
		httpx.WriteError(w, errx.New("model_missing", "mouth.glb not available; download from the mouth-glb GitHub Release", errx.ErrNotFound))
		return
	}
	w.Header().Set("Content-Type", "model/gltf-binary")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, p)
}

func mouthCandidates() []string {
	home := os.Getenv("HOME")
	return []string{
		os.Getenv("VIZ_MODEL"),
		filepath.Join("internal", "viz", "player", "static", "mouth.glb"),
		filepath.Join(home, "Desktop", "mouth.glb"),
		filepath.Join(home, "Desktop", "General", "mouth.glb"),
	}
}

func findLocalMouth() string {
	for _, p := range mouthCandidates() {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func ensureMouthModel() (string, error) {
	if p := findLocalMouth(); p != "" {
		return p, nil
	}
	mouthOnce.Do(func() {
		url := os.Getenv("VIZ_MODEL_URL")
		if url == "" {
			url = defaultMouthURL
		}
		dst := filepath.Join("internal", "viz", "player", "static", "mouth.glb")
		mouthPath, mouthErr = downloadMouth(url, dst)
	})
	return mouthPath, mouthErr
}

func downloadMouth(url, dst string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errx.New("model_missing", "mouth.glb download failed", errx.ErrNotFound)
	}
	part := dst + ".part"
	f, err := os.Create(part)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(part)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(part)
		return "", closeErr
	}
	if err := os.Rename(part, dst); err != nil {
		_ = os.Remove(part)
		return "", err
	}
	return dst, nil
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
