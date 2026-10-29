// Package domain defines share-link tokens, access rules, and patient telemetry events.
package domain

import (
	"crypto/sha256"
	"strings"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Link struct {
	ID                id.ID      `json:"id"`
	ClinicID          id.ID      `json:"clinic_id"`
	PlanID            id.ID      `json:"plan_id"`
	TokenHash         []byte     `json:"-"`
	TokenPrefix       string     `json:"token_prefix"`
	ExpiresAt         time.Time  `json:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
	ViewsCount        int        `json:"views_count"`
	LastViewedAt      *time.Time `json:"last_viewed_at,omitempty"`
	AcceptedAt        *time.Time `json:"accepted_at,omitempty"`
	AcceptedSignature string     `json:"accepted_signature,omitempty"`
	AcceptedIP        string     `json:"accepted_ip,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (l Link) IsActive(now time.Time) bool {
	if l.RevokedAt != nil {
		return false
	}
	return now.Before(l.ExpiresAt)
}

func (l Link) IsExpired(now time.Time) bool {
	return !now.Before(l.ExpiresAt)
}

func (l Link) IsRevoked() bool {
	return l.RevokedAt != nil
}

func (l Link) IsAccepted() bool {
	return l.AcceptedAt != nil
}

type EventKind string

const (
	EventOpened        EventKind = "opened"
	EventSectionViewed EventKind = "section_viewed"
	EventVizPlayed     EventKind = "viz_played"
	EventCTA           EventKind = "cta"
	EventAccepted      EventKind = "accepted"
	EventPDFDownloaded EventKind = "pdf_downloaded"
)

func ParseEventKind(raw string) EventKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(EventOpened):
		return EventOpened
	case string(EventSectionViewed):
		return EventSectionViewed
	case string(EventVizPlayed):
		return EventVizPlayed
	case string(EventCTA):
		return EventCTA
	case string(EventAccepted):
		return EventAccepted
	case string(EventPDFDownloaded):
		return EventPDFDownloaded
	default:
		return EventKind(raw)
	}
}

type Event struct {
	ID        id.ID          `json:"id"`
	ClinicID  id.ID          `json:"clinic_id"`
	LinkID    id.ID          `json:"link_id"`
	Kind      EventKind      `json:"kind"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

// HashToken calculates SHA-256 hash of a raw token.
func HashToken(raw string) []byte {
	h := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return h[:]
}
