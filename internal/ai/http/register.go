package http

import "github.com/go-chi/chi/v5"

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Register(r chi.Router) {
	// Phase 8: POST /plans/:id/ai/summary  POST /plans/:id/ai/translate
	_ = r
}
