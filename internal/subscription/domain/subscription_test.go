package domain_test

import (
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/domain"
)

func TestPlanTier_Permissions(t *testing.T) {
	classic, err := domain.ParsePlanTier("classic")
	if err != nil {
		t.Fatal(err)
	}
	if classic.AllowsUltimate() {
		t.Error("classic tier should not allow ultimate")
	}
	if !classic.AllowsClassic() {
		t.Error("classic tier should allow classic")
	}

	ultimate, err := domain.ParsePlanTier("ultimate")
	if err != nil {
		t.Fatal(err)
	}
	if !ultimate.AllowsUltimate() {
		t.Error("ultimate tier should allow ultimate")
	}
	if !ultimate.AllowsClassic() {
		t.Error("ultimate tier should allow classic")
	}

	ent, err := domain.ParsePlanTier("enterprise")
	if err != nil {
		t.Fatal(err)
	}
	if !ent.AllowsUltimate() {
		t.Error("enterprise tier should allow ultimate")
	}
}

func TestClinicSubscription_QuotaAndState(t *testing.T) {
	now := time.Now().UTC()
	sub := domain.ClinicSubscription{
		ID:          id.New(),
		WorkspaceID: id.New(),
		PlanID:      "classic_monthly",
		Plan: domain.SubscriptionPlan{
			ID:        "classic_monthly",
			Tier:      domain.TierClassic,
			PlanQuota: 30,
		},
		Status:             domain.StatusActive,
		CurrentPeriodStart: now.Add(-24 * time.Hour),
		CurrentPeriodEnd:   now.Add(29 * 24 * time.Hour),
		PlansCreatedCount:  10,
	}

	if !sub.IsActive(now) {
		t.Error("expected subscription to be active")
	}
	if sub.IsQuotaExceeded() {
		t.Error("expected quota not to be exceeded")
	}
	if sub.QuotaRemaining() != 20 {
		t.Errorf("expected 20 remaining quota, got %d", sub.QuotaRemaining())
	}
	if sub.AllowsKind("ultimate") {
		t.Error("classic subscription should not allow ultimate plan")
	}
	if !sub.AllowsKind("classic") {
		t.Error("classic subscription should allow classic plan")
	}

	// Test quota exceeded
	sub.PlansCreatedCount = 30
	if !sub.IsQuotaExceeded() {
		t.Error("expected quota to be exceeded at 30/30")
	}
	if sub.QuotaRemaining() != 0 {
		t.Errorf("expected 0 remaining quota, got %d", sub.QuotaRemaining())
	}

	// Test unlimited quota
	sub.Plan.PlanQuota = -1
	sub.Plan.Tier = domain.TierUltimate
	if sub.IsQuotaExceeded() {
		t.Error("unlimited plan should never exceed quota")
	}
	if sub.QuotaRemaining() != -1 {
		t.Errorf("expected -1 unlimited quota, got %d", sub.QuotaRemaining())
	}
	if !sub.AllowsKind("ultimate") {
		t.Error("ultimate subscription should allow ultimate plan")
	}
}
