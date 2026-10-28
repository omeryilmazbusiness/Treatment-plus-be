// Package domain is the treatment planning context (Classic form).
package domain

import (
	"strings"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type PlanKind string

const (
	KindClassic PlanKind = "classic"
)

type PlanStatus string

const (
	StatusDraft PlanStatus = "draft"
	StatusReady PlanStatus = "ready"
)

type Patient struct {
	ID          id.ID     `json:"id"`
	WorkspaceID id.ID     `json:"workspace_id"`
	Name        string    `json:"name"`
	Contact     string    `json:"contact"` // Decrypted representation in domain
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p Patient) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errx.New("invalid_patient_name", "patient name cannot be empty", errx.ErrInvalid)
	}
	return nil
}

type Visit struct {
	ID          id.ID     `json:"id"`
	PlanID      id.ID     `json:"plan_id"`
	WorkspaceID id.ID     `json:"workspace_id"`
	OrderNum    int       `json:"order_num"`
	Title       string    `json:"title"`
	PhaseLabel  string    `json:"phase_label"`
	HealingNote string    `json:"healing_note"`
	ChartTeeth  []string  `json:"chart_teeth,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Item struct {
	ID            id.ID     `json:"id"`
	PlanID        id.ID     `json:"plan_id"`
	VisitID       *id.ID    `json:"visit_id,omitempty"`
	WorkspaceID   id.ID     `json:"workspace_id"`
	TreatmentID   *id.ID    `json:"treatment_id,omitempty"`
	TreatmentName string    `json:"treatment_name"`
	ToothNumber   *string   `json:"tooth_number,omitempty"`
	Teeth         []string  `json:"teeth,omitempty"`
	Description   string    `json:"description,omitempty"`
	Quantity      int       `json:"quantity"`
	UnitPrice     int64     `json:"unit_price"` // minor units (kuruş)
	IsPackage     bool      `json:"is_package"` // package items do not increase total
	OrderNum      int       `json:"order_num"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (i Item) LineTotal() int64 {
	if i.IsPackage {
		return 0
	}
	if i.Quantity <= 0 || i.UnitPrice <= 0 {
		return 0
	}
	return int64(i.Quantity) * i.UnitPrice
}

type Discount struct {
	ID          id.ID     `json:"id"`
	PlanID      id.ID     `json:"plan_id"`
	WorkspaceID id.ID     `json:"workspace_id"`
	Label       string    `json:"label"`
	Amount      int64     `json:"amount"` // minor units (kuruş)
	OrderNum    int       `json:"order_num"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Totals struct {
	Subtotal int64 `json:"subtotal"` // minor unit
	Discount int64 `json:"discount"` // minor unit
	Grand    int64 `json:"grand"`    // minor unit
}

func CalculateTotals(items []Item, discounts []Discount, curr money.Currency) (Totals, error) {
	var subtotal int64
	for _, it := range items {
		line := it.LineTotal()
		mSub, err := money.New(subtotal, curr)
		if err != nil {
			return Totals{}, err
		}
		mLine, err := money.New(line, curr)
		if err != nil {
			return Totals{}, err
		}
		res, err := mSub.Add(mLine)
		if err != nil {
			return Totals{}, err
		}
		subtotal = res.Amount
	}

	var discountTotal int64
	for _, d := range discounts {
		if d.Amount < 0 {
			continue
		}
		mDisc, err := money.New(discountTotal, curr)
		if err != nil {
			return Totals{}, err
		}
		mAdd, err := money.New(d.Amount, curr)
		if err != nil {
			return Totals{}, err
		}
		res, err := mDisc.Add(mAdd)
		if err != nil {
			return Totals{}, err
		}
		discountTotal = res.Amount
	}

	grand := subtotal - discountTotal
	if grand < 0 {
		grand = 0
	}

	return Totals{
		Subtotal: subtotal,
		Discount: discountTotal,
		Grand:    grand,
	}, nil
}

type Plan struct {
	ID             id.ID          `json:"id"`
	WorkspaceID    id.ID          `json:"workspace_id"`
	PatientID      id.ID          `json:"patient_id"`
	Title          string         `json:"title"`
	Kind           PlanKind       `json:"kind"`
	Status         PlanStatus     `json:"status"`
	Currency       money.Currency `json:"currency"`
	Palette        Palette        `json:"palette"`
	Locale         Locale         `json:"locale"`
	Diagnoses      []string       `json:"diagnoses"`
	Situation      Situation      `json:"situation"`
	TotalsSubtotal int64          `json:"totals_subtotal"`
	TotalsDiscount int64          `json:"totals_discount"`
	TotalsGrand    int64          `json:"totals_grand"`
	SnapshotJSON   any            `json:"snapshot_json,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`

	Patient   *Patient   `json:"patient,omitempty"`
	Visits    []Visit    `json:"visits,omitempty"`
	Items     []Item     `json:"items,omitempty"`
	Discounts []Discount `json:"discounts,omitempty"`
}

type ClinicStatSnapshot struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ClinicDoctorSnapshot struct {
	Name     string               `json:"name"`
	Title    string               `json:"title"`
	Subtitle string               `json:"subtitle"`
	Bio      string               `json:"bio"`
	PhotoKey string               `json:"photo_key"`
	Stats    []ClinicStatSnapshot `json:"stats,omitempty"`
}

type ClinicLabSnapshot struct {
	Title    string               `json:"title"`
	Text     string               `json:"text"`
	ImageKey string               `json:"image_key"`
	Stats    []ClinicStatSnapshot `json:"stats,omitempty"`
}

type ClinicAwardsSnapshot struct {
	Text      string   `json:"text"`
	BadgeKeys []string `json:"badge_keys,omitempty"`
}

type ClinicHotelSnapshot struct {
	Name     string `json:"name"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key"`
}

type ClinicSnapshot struct {
	Name        string               `json:"name"`
	Slogan      string               `json:"slogan"`
	Phone       string               `json:"phone"`
	Email       string               `json:"email"`
	Website     string               `json:"website"`
	Address     string               `json:"address"`
	Hours       string               `json:"hours"`
	Palette     string               `json:"palette"`
	Locale      string               `json:"locale"`
	Currency    string               `json:"currency"`
	LogoKey     string               `json:"logo_key"`
	CoverKey    string               `json:"cover_key"`
	MapURL      string               `json:"map_url"`
	Intro       string               `json:"intro"`
	Legal       string               `json:"legal"`
	AboutText   string               `json:"about_text"`
	AboutKey    string               `json:"about_key"`
	WelcomeBody string               `json:"welcome_body"`
	SignerName  string               `json:"signer_name"`
	SignerTitle string               `json:"signer_title"`
	Highlights  []ClinicStatSnapshot `json:"highlights,omitempty"`
	Doctor      ClinicDoctorSnapshot `json:"doctor"`
	Lab         ClinicLabSnapshot    `json:"lab"`
	Awards      ClinicAwardsSnapshot `json:"awards"`
	Hotel       ClinicHotelSnapshot  `json:"hotel"`
}

type PatientSnapshot struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Contact string `json:"contact"`
}

type ItemSnapshot struct {
	VisitTitle    string   `json:"visit_title"`
	TreatmentName string   `json:"treatment_name"`
	ToothNumber   *string  `json:"tooth_number,omitempty"`
	Teeth         []string `json:"teeth,omitempty"`
	Description   string   `json:"description,omitempty"`
	Quantity      int      `json:"quantity"`
	UnitPrice     int64    `json:"unit_price"`
	IsPackage     bool     `json:"is_package"`
	LineTotal     int64    `json:"line_total"`
}

type Snapshot struct {
	PlanID    string          `json:"plan_id"`
	Title     string          `json:"title"`
	Kind      string          `json:"kind"`
	Currency  string          `json:"currency"`
	Palette   string          `json:"palette"`
	Locale    string          `json:"locale"`
	Clinic    ClinicSnapshot  `json:"clinic"`
	Patient   PatientSnapshot `json:"patient"`
	Diagnoses []string        `json:"diagnoses"`
	Situation Situation       `json:"situation"`
	Visits    []Visit         `json:"visits"`
	Items     []ItemSnapshot  `json:"items"`
	Discounts []Discount      `json:"discounts"`
	Totals    Totals          `json:"totals"`
	ReadyAt   time.Time       `json:"ready_at"`
}
