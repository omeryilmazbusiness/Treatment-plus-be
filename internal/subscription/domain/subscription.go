// Package domain implements subscription tiers, clinic licenses, usage quotas, and demo leads.
package domain

import (
	"strings"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type PlanTier string

const (
	TierClassic    PlanTier = "classic"
	TierUltimate   PlanTier = "ultimate"
	TierEnterprise PlanTier = "enterprise"
)

func ParsePlanTier(raw string) (PlanTier, error) {
	switch PlanTier(strings.ToLower(strings.TrimSpace(raw))) {
	case TierClassic:
		return TierClassic, nil
	case TierUltimate:
		return TierUltimate, nil
	case TierEnterprise:
		return TierEnterprise, nil
	default:
		return "", errx.New("invalid_tier", "tier must be classic, ultimate, or enterprise", errx.ErrInvalid)
	}
}

func (t PlanTier) AllowsUltimate() bool {
	return t == TierUltimate || t == TierEnterprise
}

func (t PlanTier) AllowsClassic() bool {
	return true
}

type BillingInterval string

const (
	IntervalMonth BillingInterval = "month"
	IntervalYear  BillingInterval = "year"
)

func ParseBillingInterval(raw string) (BillingInterval, error) {
	switch BillingInterval(strings.ToLower(strings.TrimSpace(raw))) {
	case IntervalMonth:
		return IntervalMonth, nil
	case IntervalYear:
		return IntervalYear, nil
	default:
		return "", errx.New("invalid_interval", "interval must be month or year", errx.ErrInvalid)
	}
}

type SubscriptionStatus string

const (
	StatusTrialing SubscriptionStatus = "trialing"
	StatusActive   SubscriptionStatus = "active"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusCanceled SubscriptionStatus = "canceled"
	StatusPaused   SubscriptionStatus = "paused"
)

func ParseSubscriptionStatus(raw string) (SubscriptionStatus, error) {
	switch SubscriptionStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case StatusTrialing:
		return StatusTrialing, nil
	case StatusActive:
		return StatusActive, nil
	case StatusPastDue:
		return StatusPastDue, nil
	case StatusCanceled:
		return StatusCanceled, nil
	case StatusPaused:
		return StatusPaused, nil
	default:
		return "", errx.New("invalid_status", "unknown subscription status", errx.ErrInvalid)
	}
}

// SubscriptionPlan represents a tier in the catalog (e.g. ultimate_monthly, classic_yearly).
type SubscriptionPlan struct {
	ID                   string          `json:"id"`
	Tier                 PlanTier        `json:"tier"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	BillingInterval      BillingInterval `json:"billing_interval"`
	PriceCents           int64           `json:"price_cents"`
	Currency             string          `json:"currency"`
	PlanQuota            int             `json:"plan_quota"` // -1 = unlimited
	Has3DViz             bool            `json:"has_3d_viz"`
	HasPatientPortal     bool            `json:"has_patient_portal"`
	HasTelemetry         bool            `json:"has_telemetry"`
	HasDigitalSignature  bool            `json:"has_digital_signature"`
	HasOnboardingSession bool            `json:"has_onboarding_session"`
	Features             []string        `json:"features"`
	IsActive             bool            `json:"is_active"`
	CreatedAt            time.Time       `json:"created_at"`
}

// ClinicSubscription represents the active subscription state for a tenant workspace.
type ClinicSubscription struct {
	ID                 id.ID              `json:"id"`
	WorkspaceID        id.ID              `json:"workspace_id"`
	PlanID             string             `json:"plan_id"`
	Plan               SubscriptionPlan   `json:"plan"`
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end"`
	CancelAtPeriodEnd  bool               `json:"cancel_at_period_end"`
	CanceledAt         *time.Time         `json:"canceled_at,omitempty"`
	PlansCreatedCount  int                `json:"plans_created_count"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

func (s ClinicSubscription) IsActive(now time.Time) bool {
	if s.Status != StatusActive && s.Status != StatusTrialing {
		return false
	}
	return now.Before(s.CurrentPeriodEnd)
}

func (s ClinicSubscription) AllowsKind(kind string) bool {
	k := strings.ToLower(strings.TrimSpace(kind))
	if k == "ultimate" || k == "both" {
		return s.Plan.Tier.AllowsUltimate()
	}
	return true
}

func (s ClinicSubscription) IsQuotaExceeded() bool {
	if s.Plan.PlanQuota < 0 {
		return false // unlimited
	}
	return s.PlansCreatedCount >= s.Plan.PlanQuota
}

func (s ClinicSubscription) QuotaRemaining() int {
	if s.Plan.PlanQuota < 0 {
		return -1 // unlimited
	}
	rem := s.Plan.PlanQuota - s.PlansCreatedCount
	if rem < 0 {
		return 0
	}
	return rem
}

type DemoRequest struct {
	ID          id.ID     `json:"id"`
	WebsiteURL  string    `json:"website_url"`
	ClinicName  string    `json:"clinic_name"`
	ContactName string    `json:"contact_name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Notes       string    `json:"notes"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
