// Package errx is the shared error vocabulary. HTTP adapters map these to status codes.
package errx

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrForbidden       = errors.New("forbidden")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrInvalid         = errors.New("invalid")
	ErrNotImplemented  = errors.New("not implemented")
	ErrUnavailable     = errors.New("unavailable")
)

// Error is a coded domain error. Safe to show Code to clients; Detail is internal.
type Error struct {
	Code    string
	Message string
	Detail  error
}

func (e *Error) Error() string {
	if e.Detail != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Detail }

func New(code, message string, detail error) *Error {
	return &Error{Code: code, Message: message, Detail: detail}
}

func Is(err, target error) bool { return errors.Is(err, target) }
