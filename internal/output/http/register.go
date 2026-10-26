package http

import "github.com/go-chi/chi/v5"

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Register(r chi.Router) {
	// Phase 4/5: POST /plans/:id/pdf is a command that enqueues work; GET is worker output.
	_ = r
}
