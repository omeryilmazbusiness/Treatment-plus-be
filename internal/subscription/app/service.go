package app

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/domain"
)

type Service struct {
	store SubscriptionStore
	clock clock.Clock
}

func NewService(store SubscriptionStore, clk clock.Clock) *Service {
	if clk == nil {
		clk = clock.System{}
	}
	return &Service{
		store: store,
		clock: clk,
	}
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	return s.store.ListPlans(ctx)
}

type SubscriptionSummaryDTO struct {
	ID                 string                  `json:"id"`
	PlanID             string                  `json:"plan_id"`
	Tier               string                  `json:"tier"`
	Status             string                  `json:"status"`
	BillingInterval    string                  `json:"billing_interval"`
	CurrentPeriodStart time.Time               `json:"current_period_start"`
	CurrentPeriodEnd   time.Time               `json:"current_period_end"`
	DaysRemaining      int                     `json:"days_remaining"`
	CancelAtPeriodEnd  bool                    `json:"cancel_at_period_end"`
	PlansCreatedCount  int                     `json:"plans_created_count"`
	PlanQuota          int                     `json:"plan_quota"`
	QuotaRemaining     int                     `json:"quota_remaining"`
	AllowsClassic      bool                    `json:"allows_classic"`
	AllowsUltimate     bool                    `json:"allows_ultimate"`
	Plan               domain.SubscriptionPlan `json:"plan"`
}

func (s *Service) GetCurrentSubscription(ctx context.Context, clinicID id.ID) (SubscriptionSummaryDTO, error) {
	if clinicID.IsZero() {
		return SubscriptionSummaryDTO{}, errx.New("invalid_clinic_id", "clinic id is required", errx.ErrInvalid)
	}
	sub, err := s.store.GetClinicSubscription(ctx, clinicID)
	if err != nil {
		return SubscriptionSummaryDTO{}, err
	}

	now := s.clock.Now()
	daysRem := int(math.Ceil(sub.CurrentPeriodEnd.Sub(now).Hours() / 24.0))
	if daysRem < 0 {
		daysRem = 0
	}

	return SubscriptionSummaryDTO{
		ID:                 sub.ID.String(),
		PlanID:             sub.PlanID,
		Tier:               string(sub.Plan.Tier),
		Status:             string(sub.Status),
		BillingInterval:    string(sub.Plan.BillingInterval),
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		DaysRemaining:      daysRem,
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
		PlansCreatedCount:  sub.PlansCreatedCount,
		PlanQuota:          sub.Plan.PlanQuota,
		QuotaRemaining:     sub.QuotaRemaining(),
		AllowsClassic:      sub.Plan.Tier.AllowsClassic(),
		AllowsUltimate:     sub.Plan.Tier.AllowsUltimate(),
		Plan:               sub.Plan,
	}, nil
}

func (s *Service) Subscribe(ctx context.Context, clinicID id.ID, planID string) (SubscriptionSummaryDTO, error) {
	if clinicID.IsZero() {
		return SubscriptionSummaryDTO{}, errx.New("invalid_clinic_id", "clinic id is required", errx.ErrInvalid)
	}
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return SubscriptionSummaryDTO{}, errx.New("invalid_plan_id", "plan_id is required", errx.ErrInvalid)
	}

	plan, err := s.store.GetPlan(ctx, planID)
	if err != nil {
		return SubscriptionSummaryDTO{}, err
	}
	if !plan.IsActive {
		return SubscriptionSummaryDTO{}, errx.New("plan_inactive", "selected plan is not active", errx.ErrInvalid)
	}

	now := s.clock.Now()
	periodEnd := now.AddDate(0, 1, 0)
	if plan.BillingInterval == domain.IntervalYear {
		periodEnd = now.AddDate(1, 0, 0)
	}

	sub := domain.ClinicSubscription{
		ID:                 id.New(),
		WorkspaceID:        clinicID,
		PlanID:             plan.ID,
		Plan:               plan,
		Status:             domain.StatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		CancelAtPeriodEnd:  false,
		PlansCreatedCount:  0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.store.UpsertClinicSubscription(ctx, sub); err != nil {
		return SubscriptionSummaryDTO{}, err
	}

	return s.GetCurrentSubscription(ctx, clinicID)
}

func (s *Service) ChangeTier(ctx context.Context, clinicID id.ID, newPlanID string) (SubscriptionSummaryDTO, error) {
	if clinicID.IsZero() {
		return SubscriptionSummaryDTO{}, errx.New("invalid_clinic_id", "clinic id is required", errx.ErrInvalid)
	}
	newPlanID = strings.TrimSpace(newPlanID)
	if newPlanID == "" {
		return SubscriptionSummaryDTO{}, errx.New("invalid_plan_id", "new plan_id is required", errx.ErrInvalid)
	}

	current, err := s.store.GetClinicSubscription(ctx, clinicID)
	if err != nil {
		return SubscriptionSummaryDTO{}, err
	}

	targetPlan, err := s.store.GetPlan(ctx, newPlanID)
	if err != nil {
		return SubscriptionSummaryDTO{}, err
	}
	if !targetPlan.IsActive {
		return SubscriptionSummaryDTO{}, errx.New("plan_inactive", "target plan is not active", errx.ErrInvalid)
	}

	now := s.clock.Now()
	periodEnd := now.AddDate(0, 1, 0)
	if targetPlan.BillingInterval == domain.IntervalYear {
		periodEnd = now.AddDate(1, 0, 0)
	}

	current.PlanID = targetPlan.ID
	current.Plan = targetPlan
	current.Status = domain.StatusActive
	current.CurrentPeriodStart = now
	current.CurrentPeriodEnd = periodEnd
	current.CancelAtPeriodEnd = false
	current.UpdatedAt = now

	if err := s.store.UpsertClinicSubscription(ctx, current); err != nil {
		return SubscriptionSummaryDTO{}, err
	}

	return s.GetCurrentSubscription(ctx, clinicID)
}

func (s *Service) CancelSubscription(ctx context.Context, clinicID id.ID) (SubscriptionSummaryDTO, error) {
	if clinicID.IsZero() {
		return SubscriptionSummaryDTO{}, errx.New("invalid_clinic_id", "clinic id is required", errx.ErrInvalid)
	}

	current, err := s.store.GetClinicSubscription(ctx, clinicID)
	if err != nil {
		return SubscriptionSummaryDTO{}, err
	}

	now := s.clock.Now()
	current.CancelAtPeriodEnd = true
	current.CanceledAt = &now
	current.UpdatedAt = now

	if err := s.store.UpsertClinicSubscription(ctx, current); err != nil {
		return SubscriptionSummaryDTO{}, err
	}

	return s.GetCurrentSubscription(ctx, clinicID)
}

// CanCreatePlan verifies that the clinic has an active subscription, allows the requested plan kind,
// and has not exhausted their period quota.
func (s *Service) CanCreatePlan(ctx context.Context, clinicID id.ID, kind string) error {
	if clinicID.IsZero() {
		return errx.New("invalid_clinic_id", "clinic id is required", errx.ErrInvalid)
	}
	sub, err := s.store.GetClinicSubscription(ctx, clinicID)
	if err != nil {
		return errx.New("subscription_required", "active subscription required to create treatment plans", errx.ErrForbidden)
	}

	now := s.clock.Now()
	if !sub.IsActive(now) {
		return errx.New("subscription_inactive", "your subscription is expired or inactive; please renew to create plans", errx.ErrForbidden)
	}

	if !sub.AllowsKind(kind) {
		return errx.New("upgrade_required", "ultimate subscription tier required for interactive 3D presentations", errx.ErrForbidden)
	}

	if sub.IsQuotaExceeded() {
		return errx.New("quota_exceeded", "monthly treatment plan quota reached for this billing period", errx.ErrForbidden)
	}

	return nil
}

func (s *Service) RecordPlanUsage(ctx context.Context, clinicID id.ID) error {
	if clinicID.IsZero() {
		return errx.New("invalid_clinic_id", "clinic id is required", errx.ErrInvalid)
	}
	return s.store.IncrementPlanCount(ctx, clinicID)
}

type SubmitDemoInput struct {
	WebsiteURL  string `json:"website_url"`
	ClinicName  string `json:"clinic_name"`
	ContactName string `json:"contact_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Notes       string `json:"notes"`
}

func (s *Service) SubmitDemoRequest(ctx context.Context, in SubmitDemoInput) (domain.DemoRequest, error) {
	url := strings.TrimSpace(in.WebsiteURL)
	if url == "" {
		return domain.DemoRequest{}, errx.New("website_url_required", "website url is required to design demo", errx.ErrInvalid)
	}

	req := domain.DemoRequest{
		ID:          id.New(),
		WebsiteURL:  url,
		ClinicName:  strings.TrimSpace(in.ClinicName),
		ContactName: strings.TrimSpace(in.ContactName),
		Email:       strings.TrimSpace(in.Email),
		Phone:       strings.TrimSpace(in.Phone),
		Notes:       strings.TrimSpace(in.Notes),
		Status:      "pending",
		CreatedAt:   s.clock.Now(),
	}

	if err := s.store.CreateDemoRequest(ctx, req); err != nil {
		return domain.DemoRequest{}, err
	}

	return req, nil
}

