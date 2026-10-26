package httpx

import (
	"errors"
	"net/http"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteError maps domain errors to HTTP status. Handlers stay thin.
func WriteError(w http.ResponseWriter, err error) {
	status, code, message := http.StatusInternalServerError, "internal", "internal error"
	switch {
	case errors.Is(err, errx.ErrNotFound):
		status, code, message = http.StatusNotFound, "not_found", "not found"
	case errors.Is(err, errx.ErrUnauthenticated):
		status, code, message = http.StatusUnauthorized, "unauthenticated", "unauthenticated"
	case errors.Is(err, errx.ErrForbidden):
		status, code, message = http.StatusForbidden, "forbidden", "forbidden"
	case errors.Is(err, errx.ErrConflict):
		status, code, message = http.StatusConflict, "conflict", "conflict"
	case errors.Is(err, errx.ErrInvalid):
		status, code, message = http.StatusUnprocessableEntity, "invalid", "invalid request"
	case errors.Is(err, errx.ErrNotImplemented):
		status, code, message = http.StatusNotImplemented, "not_implemented", "not implemented"
	case errors.Is(err, errx.ErrUnavailable):
		status, code, message = http.StatusServiceUnavailable, "unavailable", "unavailable"
	}

	var coded *errx.Error
	if errors.As(err, &coded) && coded.Code != "" {
		code = coded.Code
		if coded.Message != "" {
			message = coded.Message
		}
	}

	writeJSON(w, status, errorBody{Error: errorPayload{Code: code, Message: message}})
}
