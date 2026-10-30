package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/app"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/mem"
)

func TestSubscriptionService_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	store := mem.New()
	clk := clock.System{}
	svc := app.NewService(store, clk)

	clinicID := id.New()

	// 1. List Plans
	plans, err := svc.ListPlans(ctx)
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(plans) < 4 {
		t.Fatalf("expected at least 4 plans, got %d", len(plans))
	}

	// 2. Unsubscribed clinic cannot create plans
	err = svc.CanCreatePlan(ctx, clinicID, "classic")
	if err == nil || !errors.Is(err, errx.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for unsubscribed clinic, got: %v", err)
	}

	// 3. Subscribe to Classic Monthly
	sub, err := svc.Subscribe(ctx, clinicID, "classic_monthly")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if sub.Tier != "classic" {
		t.Fatalf("expected tier classic, got %s", sub.Tier)
	}
	if sub.PlanQuota != 30 {
		t.Fatalf("expected plan quota 30, got %d", sub.PlanQuota)
	}

	// 4. Classic tier can create classic plan, but CANNOT create ultimate plan
	if err := svc.CanCreatePlan(ctx, clinicID, "classic"); err != nil {
		t.Fatalf("expected classic plan to be permitted, got: %v", err)
	}
	err = svc.CanCreatePlan(ctx, clinicID, "ultimate")
	if err == nil || !errors.Is(err, errx.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for ultimate on classic tier, got: %v", err)
	}

	// 5. Upgrade to Ultimate Yearly (change tier)
	upgraded, err := svc.ChangeTier(ctx, clinicID, "ultimate_yearly")
	if err != nil {
		t.Fatalf("ChangeTier failed: %v", err)
	}
	if upgraded.Tier != "ultimate" {
		t.Fatalf("expected tier ultimate, got %s", upgraded.Tier)
	}
	if upgraded.BillingInterval != "year" {
		t.Fatalf("expected interval year, got %s", upgraded.BillingInterval)
	}
	if upgraded.PlanQuota != -1 {
		t.Fatalf("expected unlimited quota (-1), got %d", upgraded.PlanQuota)
	}

	// 6. Now Ultimate plan is permitted
	if err := svc.CanCreatePlan(ctx, clinicID, "ultimate"); err != nil {
		t.Fatalf("expected ultimate plan to be permitted on ultimate tier, got: %v", err)
	}

	// 7. Record plan usage
	if err := svc.RecordPlanUsage(ctx, clinicID); err != nil {
		t.Fatalf("RecordPlanUsage failed: %v", err)
	}

	cur, err := svc.GetCurrentSubscription(ctx, clinicID)
	if err != nil {
		t.Fatalf("GetCurrentSubscription failed: %v", err)
	}
	if cur.PlansCreatedCount != 1 {
		t.Fatalf("expected plans created count 1, got %d", cur.PlansCreatedCount)
	}

	// 8. Submit Demo Request
	demo, err := svc.SubmitDemoRequest(ctx, app.SubmitDemoInput{
		WebsiteURL:  "https://dentsmile.com",
		ClinicName:  "DentSmile",
		ContactName: "Dr. Alex",
		Email:       "alex@dentsmile.com",
		Phone:       "+447700900123",
	})
	if err != nil {
		t.Fatalf("SubmitDemoRequest failed: %v", err)
	}
	if demo.WebsiteURL != "https://dentsmile.com" {
		t.Fatalf("expected website url https://dentsmile.com, got %s", demo.WebsiteURL)
	}

	// 9. Cancel subscription (sets cancel_at_period_end)
	canceled, err := svc.CancelSubscription(ctx, clinicID)
	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}
	if !canceled.CancelAtPeriodEnd {
		t.Fatal("expected cancel_at_period_end to be true")
	}
}
