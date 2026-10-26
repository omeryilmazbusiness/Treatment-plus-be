package http

import "github.com/go-chi/chi/v5"

// Handler is the identity HTTP adapter. Wired in the composition root.
type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Register(r chi.Router) {
	// Phase 1:
	// POST /auth/register
	// POST /auth/login
	// POST /auth/refresh
	// POST /auth/logout
	// POST /auth/invite
	_ = r
}
