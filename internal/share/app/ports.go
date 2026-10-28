package app

import (
	"context"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type ShareStore interface {
	CreateLink(ctx context.Context, link domain.Link) error
	GetLink(ctx context.Context, clinicID, linkID id.ID) (domain.Link, error)
	ListLinks(ctx context.Context, clinicID, planID id.ID) ([]domain.Link, error)
	RevokeLink(ctx context.Context, clinicID, linkID id.ID) error

	// Public unauthenticated methods (executed via SECURITY DEFINER functions)
	ResolveToken(ctx context.Context, tokenHash []byte) (domain.Link, error)
	RecordEvent(ctx context.Context, tokenHash []byte, eventID id.ID, kind domain.EventKind, metadata map[string]any) error
	AcceptPlan(ctx context.Context, tokenHash []byte, signature, ip string) error
	ListEvents(ctx context.Context, clinicID, planID id.ID) ([]domain.Event, error)
}

type SharedItem struct {
	ID            string   `json:"id"`
	VisitID       *string  `json:"visit_id,omitempty"`
	TreatmentName string   `json:"treatment_name"`
	AnimationCode string   `json:"animation_code,omitempty"`
	ToothNumber   *string  `json:"tooth_number,omitempty"`
	Teeth         []string `json:"teeth,omitempty"`
	Description   string   `json:"description,omitempty"`
	Quantity      int      `json:"quantity"`
	UnitPrice     int64    `json:"unit_price"`
	IsPackage     bool     `json:"is_package"`
}

type SharedVisit struct {
	ID          string       `json:"id"`
	OrderNum    int          `json:"order_num"`
	Title       string       `json:"title"`
	PhaseLabel  string       `json:"phase_label,omitempty"`
	HealingNote string       `json:"healing_note,omitempty"`
	ChartTeeth  []string     `json:"chart_teeth,omitempty"`
	Items       []SharedItem `json:"items,omitempty"`
}

type SharedDiscount struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

type SharedToothFinding struct {
	Tooth  string `json:"tooth"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

type SharedSituation struct {
	XrayKey          string               `json:"xray_key,omitempty"`
	ChartTeeth       []string             `json:"chart_teeth,omitempty"`
	ToothFindings    []SharedToothFinding `json:"tooth_findings,omitempty"`
	GeneralDiagnoses []string             `json:"general_diagnoses,omitempty"`
}

type SharedPlanData struct {
	ID                 id.ID            `json:"id"`
	ClinicID           id.ID            `json:"clinic_id"`
	PatientName        string           `json:"patient_name"`
	Title              string           `json:"title"`
	Kind               string           `json:"kind"`
	Status             string           `json:"status"`
	Currency           string           `json:"currency"`
	Palette            string           `json:"palette"`
	Locale             string           `json:"locale"`
	WelcomeLetter      string           `json:"welcome_letter"`
	VideoURL           string           `json:"video_url"`
	BeforeAfterCaseIDs []string         `json:"before_after_case_ids"`
	Diagnoses          []string         `json:"diagnoses"`
	Situation          SharedSituation  `json:"situation"`
	Visits             []SharedVisit    `json:"visits"`
	Items              []SharedItem     `json:"items"`
	Discounts          []SharedDiscount `json:"discounts"`
	TotalsSubtotal     int64            `json:"totals_subtotal"`
	TotalsDiscount     int64            `json:"totals_discount"`
	TotalsGrand        int64            `json:"totals_grand"`
	CreatedAt          time.Time        `json:"created_at"`
}

type SharedStat struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type SharedDoctor struct {
	Name     string       `json:"name"`
	Title    string       `json:"title"`
	Subtitle string       `json:"subtitle,omitempty"`
	Bio      string       `json:"bio,omitempty"`
	PhotoKey string       `json:"photo_key,omitempty"`
	Stats    []SharedStat `json:"stats,omitempty"`
}

type SharedLab struct {
	Title    string       `json:"title"`
	Text     string       `json:"text"`
	ImageKey string       `json:"image_key,omitempty"`
	Stats    []SharedStat `json:"stats,omitempty"`
}

type SharedAwards struct {
	Text      string   `json:"text"`
	BadgeKeys []string `json:"badge_keys,omitempty"`
}

type SharedHotel struct {
	Name     string `json:"name"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key,omitempty"`
}

type SharedReview struct {
	Author   string `json:"author"`
	Rating   int    `json:"rating"`
	Text     string `json:"text"`
	Source   string `json:"source"`
	Date     string `json:"date,omitempty"`
	Verified bool   `json:"verified"`
}

type SharedBeforeAfterCase struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Tag            string `json:"tag"`
	Description    string `json:"description"`
	BeforeImageKey string `json:"before_image_key"`
	AfterImageKey  string `json:"after_image_key"`
}

type SharedCTAConfig struct {
	WhatsAppNumber   string `json:"whatsapp_number"`
	WhatsAppMessage  string `json:"whatsapp_message"`
	PhoneNumber      string `json:"phone_number"`
	BookingURL       string `json:"booking_url"`
	EnableAcceptance bool   `json:"enable_acceptance"`
}

type SharedClinicData struct {
	Name        string                  `json:"name"`
	Slogan      string                  `json:"slogan"`
	Phone       string                  `json:"phone"`
	Email       string                  `json:"email"`
	Website     string                  `json:"website"`
	Address     string                  `json:"address"`
	Hours       string                  `json:"hours"`
	LogoKey     string                  `json:"logo_key"`
	CoverKey    string                  `json:"cover_key"`
	MapURL      string                  `json:"map_url"`
	Intro       string                  `json:"intro"`
	Legal       string                  `json:"legal"`
	AboutText   string                  `json:"about_text"`
	AboutKey    string                  `json:"about_key"`
	WelcomeBody string                  `json:"welcome_body"`
	SignerName  string                  `json:"signer_name"`
	SignerTitle string                  `json:"signer_title"`
	Highlights  []SharedStat            `json:"highlights"`
	Doctor      SharedDoctor            `json:"doctor"`
	Lab         SharedLab               `json:"lab"`
	Awards      SharedAwards            `json:"awards"`
	Hotel       SharedHotel             `json:"hotel"`
	Team        []SharedDoctor          `json:"team"`
	Reviews     []SharedReview          `json:"reviews"`
	BeforeAfter []SharedBeforeAfterCase `json:"before_after"`
	CTA         SharedCTAConfig         `json:"cta"`
}

type SharedVizClip struct {
	Code  string   `json:"code"`
	Teeth []string `json:"teeth"`
	Title string   `json:"title"`
	Moves any      `json:"moves"`
}

type SharedVizPlaylist struct {
	PlanID string          `json:"plan_id"`
	Locale string          `json:"locale"`
	Clips  []SharedVizClip `json:"clips"`
}

type PlanReader interface {
	GetPlanForShare(ctx context.Context, clinicID, planID id.ID) (SharedPlanData, error)
}

type ClinicReader interface {
	GetClinicBrand(ctx context.Context, clinicID id.ID) (SharedClinicData, error)
}

type VizReader interface {
	GetVizPlaylist(ctx context.Context, clinicID, planID id.ID) (SharedVizPlaylist, error)
}

type PDFReader interface {
	GetPlanPDF(ctx context.Context, clinicID, planID id.ID) ([]byte, error)
}
