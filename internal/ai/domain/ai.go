// Package domain is AI draft jobs. No auto-send. The clinician always edits first.
package domain

import "github.com/treatment-plus/treatment-plus-be/internal/shared/id"

type Kind string

const (
	KindSummary   Kind = "summary"
	KindLetter    Kind = "letter"
	KindTranslate Kind = "translate"
)

type Job struct {
	ID     id.ID
	PlanID id.ID
	Kind   Kind
}
