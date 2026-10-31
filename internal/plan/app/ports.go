package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type ClinicStatDTO struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ClinicDoctorDTO struct {
	Name     string          `json:"name"`
	Title    string          `json:"title"`
	Subtitle string          `json:"subtitle"`
	Bio      string          `json:"bio"`
	PhotoKey string          `json:"photo_key"`
	Stats    []ClinicStatDTO `json:"stats,omitempty"`
}

type ClinicLabDTO struct {
	Title    string          `json:"title"`
	Text     string          `json:"text"`
	ImageKey string          `json:"image_key"`
	Stats    []ClinicStatDTO `json:"stats,omitempty"`
}

type ClinicAwardsDTO struct {
	Text      string   `json:"text"`
	BadgeKeys []string `json:"badge_keys,omitempty"`
}

type ClinicHotelDTO struct {
	Name     string `json:"name"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key"`
}

type ClinicSettingsDTO struct {
	Name        string          `json:"name"`
	Slogan      string          `json:"slogan"`
	Phone       string          `json:"phone"`
	Email       string          `json:"email"`
	Website     string          `json:"website"`
	Address     string          `json:"address"`
	Hours       string          `json:"hours"`
	Palette     string          `json:"palette"`
	Locale      string          `json:"locale"`
	Currency    string          `json:"currency"`
	LogoKey     string          `json:"logo_key"`
	CoverKey    string          `json:"cover_key"`
	MapURL      string          `json:"map_url"`
	Intro       string          `json:"intro"`
	Legal       string          `json:"legal"`
	AboutText   string          `json:"about_text"`
	AboutKey    string          `json:"about_key"`
	WelcomeBody string          `json:"welcome_body"`
	SignerName  string          `json:"signer_name"`
	SignerTitle string          `json:"signer_title"`
	Highlights  []ClinicStatDTO `json:"highlights,omitempty"`
	Doctor      ClinicDoctorDTO `json:"doctor"`
	Lab         ClinicLabDTO    `json:"lab"`
	Awards      ClinicAwardsDTO `json:"awards"`
	Hotel       ClinicHotelDTO  `json:"hotel"`
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

type SubscriptionEnforcer interface {
	CanCreatePlan(ctx context.Context, wsID id.ID, kind string) error
	RecordPlanUsage(ctx context.Context, wsID id.ID) error
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
	ReopenDraft(ctx context.Context, wsID id.ID, planID id.ID) error
}
