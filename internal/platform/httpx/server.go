package httpx

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
)

// RouteRegistrar is implemented by each context's HTTP adapter.
type RouteRegistrar interface {
	Register(r chi.Router)
}

type Server struct {
	HTTP   *http.Server
	Router chi.Router
}

type Options struct {
	Config     config.Config
	Log        *slog.Logger
	Ready      []Readiness
	Registrars []RouteRegistrar
}

func NewServer(opts Options) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestID)
	r.Use(withLogger(opts.Log))
	r.Use(accessLog)
	r.Use(securityHeaders)
	r.Use(maxBytes(opts.Config.HTTP.MaxBodyBytes))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   opts.Config.HTTP.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id", "X-Requested-With", "X-Internal-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", handleHealthz)
	r.Get("/readyz", handleReadyz(opts.Ready))
	r.Get("/workspace", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"products": []map[string]any{
				{"id": "classic", "open": true},
				{"id": "ultimate", "open": false},
			},
		})
	})

	r.Route("/v1", func(v1 chi.Router) {
		for _, reg := range opts.Registrars {
			if reg != nil {
				reg.Register(v1)
			}
		}
	})

	return &Server{
		Router: r,
		HTTP: &http.Server{
			Addr:         opts.Config.HTTP.Addr,
			Handler:      r,
			ReadTimeout:  opts.Config.HTTP.ReadTimeout,
			WriteTimeout: opts.Config.HTTP.WriteTimeout,
			IdleTimeout:  opts.Config.HTTP.IdleTimeout,
		},
	}
}
