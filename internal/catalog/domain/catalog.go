// Package domain is the diagnosis/treatment/price-list catalog.
// Sent plans are immutable snapshots; this context never mutates them.
package domain

import (
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Diagnosis struct {
	ID   id.ID
	Code string
	Name string
}

type Treatment struct {
	ID          id.ID
	Code        string
	Name        string
	NeedsTooth  bool
	DraftCopy   string
}

type PriceItem struct {
	TreatmentID id.ID
	UnitPrice   money.Money
}

type PriceList struct {
	ID    id.ID
	Name  string
	Items []PriceItem
}

type Package struct {
	ID      id.ID
	Name    string
	ItemIDs []id.ID
}
