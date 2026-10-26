// Package domain is the plan engine: patient, visits, lines, totals, state machine.
// No PDF, AI, or HTTP types live here.
package domain

import (
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusReady    Status = "ready"
	StatusSent     Status = "sent"
	StatusViewed   Status = "viewed"
	StatusAccepted Status = "accepted"
	StatusExpired  Status = "expired"
	StatusRevoked  Status = "revoked"
)

type Patient struct {
	ID       id.ID
	ClinicID id.ID
	Name     string
}

type Visit struct {
	ID       id.ID
	Ordinal  int
	Title    string
	WaitNote string
	Items    []Item
}

type Item struct {
	ID          id.ID
	TreatmentID id.ID
	Quantity    int64
	UnitPrice   money.Money
	InPackage   bool
}

type Discount struct {
	Label  string
	Amount money.Money
}

type Totals struct {
	Subtotal money.Money
	Discount money.Money
	Grand    money.Money
}

type Plan struct {
	ID         id.ID
	ClinicID   id.ID
	PatientID  id.ID
	Status     Status
	Visits     []Visit
	Discounts  []Discount
	Totals     Totals
	Snapshot   []byte
}
