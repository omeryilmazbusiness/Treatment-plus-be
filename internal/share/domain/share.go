// Package domain is share-link tokens and events. Tokens are stored hashed.
package domain

import (
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Link struct {
	ID        id.ID
	ClinicID  id.ID
	PlanID    id.ID
	TokenHash []byte
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type EventKind string

const (
	EventOpened   EventKind = "opened"
	EventCTA      EventKind = "cta"
	EventAccepted EventKind = "accepted"
)

type Event struct {
	ID     id.ID
	LinkID id.ID
	Kind   EventKind
}
