package http

import "github.com/go-chi/chi/v5"

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Register(r chi.Router) {
	// Phase 4: /patients  /plans  PUT .../visits|items|odontogram
	// POST /plans/:id/ready
	_ = r
}
