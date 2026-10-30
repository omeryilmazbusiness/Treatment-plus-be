package mem

import (
	"context"
	"sync"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/domain"
)

type Store struct {
	mu           sync.RWMutex
	plans        map[string]domain.SubscriptionPlan
	subs         map[id.ID]domain.ClinicSubscription
	demoRequests []domain.DemoRequest
}

func New() *Store {
	s := &Store{
		plans: make(map[string]domain.SubscriptionPlan),
		subs:  make(map[id.ID]domain.ClinicSubscription),
	}
	// Seed default plans
	now := time.Now().UTC()
	s.plans["classic_monthly"] = domain.SubscriptionPlan{
		ID:              "classic_monthly",
		Tier:            domain.TierClassic,
		Name:            "BrightPlans Classic (Monthly)",
		BillingInterval: domain.IntervalMonth,
		PriceCents:      4900,
		Currency:        "EUR",
		PlanQuota:       30,
		IsActive:        true,
		CreatedAt:       now,
	}
	s.plans["classic_yearly"] = domain.SubscriptionPlan{
		ID:              "classic_yearly",
		Tier:            domain.TierClassic,
		Name:            "BrightPlans Classic (Yearly)",
		BillingInterval: domain.IntervalYear,
		PriceCents:      49000,
		Currency:        "EUR",
		PlanQuota:       50,
		IsActive:        true,
		CreatedAt:       now,
	}
	s.plans["ultimate_monthly"] = domain.SubscriptionPlan{
		ID:                  "ultimate_monthly",
		Tier:                domain.TierUltimate,
		Name:                "BrightPlans Ultimate (Monthly)",
		BillingInterval:     domain.IntervalMonth,
		PriceCents:          9900,
		Currency:            "EUR",
		PlanQuota:           100,
		Has3DViz:            true,
		HasPatientPortal:    true,
		HasTelemetry:        true,
		HasDigitalSignature: true,
		IsActive:            true,
		CreatedAt:           now,
	}
	s.plans["ultimate_yearly"] = domain.SubscriptionPlan{
		ID:                   "ultimate_yearly",
		Tier:                 domain.TierUltimate,
		Name:                 "BrightPlans Ultimate (Yearly)",
		BillingInterval:      domain.IntervalYear,
		PriceCents:           99000,
		Currency:             "EUR",
		PlanQuota:            -1,
		Has3DViz:             true,
		HasPatientPortal:     true,
		HasTelemetry:         true,
		HasDigitalSignature:  true,
		HasOnboardingSession: true,
		IsActive:             true,
		CreatedAt:            now,
	}
	return s
}

func (s *Store) ListPlans(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.SubscriptionPlan, 0, len(s.plans))
	for _, p := range s.plans {
		out = append(out, p)
	}
	return out, nil
}

func (s *Store) GetPlan(ctx context.Context, planID string) (domain.SubscriptionPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.plans[planID]
	if !ok {
		return domain.SubscriptionPlan{}, errx.New("plan_not_found", "subscription plan not found", errx.ErrNotFound)
	}
	return p, nil
}

func (s *Store) GetClinicSubscription(ctx context.Context, clinicID id.ID) (domain.ClinicSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, ok := s.subs[clinicID]
	if !ok {
		return domain.ClinicSubscription{}, errx.New("subscription_not_found", "clinic has no active subscription", errx.ErrNotFound)
	}
	return sub, nil
}

func (s *Store) UpsertClinicSubscription(ctx context.Context, sub domain.ClinicSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.subs[sub.WorkspaceID] = sub
	return nil
}

func (s *Store) IncrementPlanCount(ctx context.Context, clinicID id.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subs[clinicID]
	if !ok {
		return errx.New("subscription_not_found", "clinic has no active subscription", errx.ErrNotFound)
	}
	sub.PlansCreatedCount++
	s.subs[clinicID] = sub
	return nil
}

func (s *Store) CreateDemoRequest(ctx context.Context, req domain.DemoRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.demoRequests = append(s.demoRequests, req)
	return nil
}

func (s *Store) ListDemoRequests(ctx context.Context) ([]domain.DemoRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.DemoRequest, len(s.demoRequests))
	copy(out, s.demoRequests)
	return out, nil
}

