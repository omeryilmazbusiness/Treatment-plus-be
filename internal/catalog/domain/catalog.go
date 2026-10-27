// Package domain is the treatment & pricing catalog context.
package domain

import (
	"strings"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Diagnosis struct {
	ID          id.ID     `json:"id"`
	WorkspaceID id.ID     `json:"workspace_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (d Diagnosis) Validate() error {
	if strings.TrimSpace(d.Code) == "" {
		return errx.New("invalid_diagnosis_code", "diagnosis code cannot be empty", errx.ErrInvalid)
	}
	if strings.TrimSpace(d.Name) == "" {
		return errx.New("invalid_diagnosis_name", "diagnosis name cannot be empty", errx.ErrInvalid)
	}
	return nil
}

type Treatment struct {
	ID          id.ID     `json:"id"`
	WorkspaceID id.ID     `json:"workspace_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (t Treatment) Validate() error {
	if strings.TrimSpace(t.Code) == "" {
		return errx.New("invalid_treatment_code", "treatment code cannot be empty", errx.ErrInvalid)
	}
	if strings.TrimSpace(t.Name) == "" {
		return errx.New("invalid_treatment_name", "treatment name cannot be empty", errx.ErrInvalid)
	}
	return nil
}

type PriceList struct {
	ID          id.ID          `json:"id"`
	WorkspaceID id.ID          `json:"workspace_id"`
	Name        string         `json:"name"`
	Currency    money.Currency `json:"currency"`
	IsDefault   bool           `json:"is_default"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (p PriceList) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errx.New("invalid_price_list_name", "price list name cannot be empty", errx.ErrInvalid)
	}
	if !money.ValidCurrency(p.Currency) {
		return errx.New("invalid_currency", "currency must be TRY, EUR, or USD", errx.ErrInvalid)
	}
	return nil
}

type PriceItem struct {
	ID          id.ID     `json:"id"`
	WorkspaceID id.ID     `json:"workspace_id"`
	PriceListID id.ID     `json:"price_list_id"`
	TreatmentID id.ID     `json:"treatment_id"`
	Amount      int64     `json:"amount"` // Integer minor unit (kuruş)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (i PriceItem) Validate() error {
	if i.Amount < 0 {
		return errx.New("invalid_price_item_amount", "amount cannot be negative", errx.ErrInvalid)
	}
	return nil
}
