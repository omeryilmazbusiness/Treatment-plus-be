package http

import "github.com/go-chi/chi/v5"

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Register(r chi.Router) {
	// Phase 6: POST /plans/:id/send
	// GET  /public/plans/:token
	// POST /public/plans/:token/events
	_ = r
}
