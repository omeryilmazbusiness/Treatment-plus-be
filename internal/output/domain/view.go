// Package domain is the read-only PlanView DTO shared by web and Classic PDF.
// Output never mutates a plan.
package domain

import "github.com/treatment-plus/treatment-plus-be/internal/shared/id"

// PlanView is the single output model. Web and PDF render this; a second price model is forbidden.
type PlanView struct {
	PlanID   id.ID
	ClinicID id.ID
	Payload  []byte
}
