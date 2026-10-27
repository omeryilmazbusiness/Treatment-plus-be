package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type ClinicSettingsDTO struct {
	Name    string
	Phone   string
	Palette string
	LogoKey string
	Intro   string
	Legal   string
}

type SettingsReader interface {
	GetClinicSettings(ctx context.Context, wsID id.ID) (ClinicSettingsDTO, error)
}

type CatalogReader interface {
	GetTreatment(ctx context.Context, wsID id.ID, treatID id.ID) (code string, name string, price int64, err error)
}

type Encrypter interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type PatientStore interface {
	ListPatients(ctx context.Context, wsID id.ID) ([]domain.Patient, error)
	GetPatient(ctx context.Context, wsID id.ID, patientID id.ID) (domain.Patient, error)
	SavePatient(ctx context.Context, p domain.Patient, encryptedContact string) error
}

type PlanStore interface {
	ListPlans(ctx context.Context, wsID id.ID) ([]domain.Plan, error)
	GetPlan(ctx context.Context, wsID id.ID, planID id.ID) (domain.Plan, error)
	CreatePlan(ctx context.Context, p domain.Plan) error
	UpdatePlanHeader(ctx context.Context, p domain.Plan) error
	ReplaceVisits(ctx context.Context, wsID id.ID, planID id.ID, visits []domain.Visit) error
	ReplaceItems(ctx context.Context, wsID id.ID, planID id.ID, items []domain.Item, totals domain.Totals) error
	ReplaceDiscounts(ctx context.Context, wsID id.ID, planID id.ID, discounts []domain.Discount, totals domain.Totals) error
	SetReady(ctx context.Context, wsID id.ID, planID id.ID, snapshotJSON []byte) error
}
