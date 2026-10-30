package pg_test

import (
	"context"
	"errors"
	"testing"
	"time"

	clinicapp "github.com/treatment-plus/treatment-plus-be/internal/clinic/app"
	clinicpg "github.com/treatment-plus/treatment-plus-be/internal/clinic/pg"
	identityapp "github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	identitypg "github.com/treatment-plus/treatment-plus-be/internal/identity/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres/pgtest"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/domain"
	subpg "github.com/treatment-plus/treatment-plus-be/internal/subscription/pg"
)

func TestSubscriptionPG_LifecycleAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.Open(t)

	clk := clock.System{}
	tokens, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-access-secret-32bytes-ok!",
		AccessTTL:       time.Minute,
	}, clk)
	if err != nil {
		t.Fatal(err)
	}

	idStore := identitypg.New(pool)
	clinicStore := clinicpg.New(pool)
	clinicSvc := clinicapp.NewService(clinicStore)

	idSvc, err := identityapp.NewService(idStore, idStore, idStore, idStore, authn.FastArgon2id(), tokens, clk, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}

	suffix := id.New().String()
	regA, err := idSvc.Register(ctx, identityapp.RegisterInput{
		Name: "Clinic Sub A", Email: "sub-a-" + suffix + "@test.com", Password: "password123", Role: "clinic",
		Clinic: identityapp.RequiredClinic("+90 555 111 2233"),
	})
	if err != nil {
		t.Fatal(err)
	}
	clinicA := regA.Account.Workspace.ID

	regB, err := idSvc.Register(ctx, identityapp.RegisterInput{
		Name: "Clinic Sub B", Email: "sub-b-" + suffix + "@test.com", Password: "password123", Role: "clinic",
		Clinic: identityapp.RequiredClinic("+90 555 999 8877"),
	})
	if err != nil {
		t.Fatal(err)
	}
	clinicB := regB.Account.Workspace.ID
	_ = clinicSvc

	store := subpg.New(pool)

	// 1. List Plans (Public)
	plans, err := store.ListPlans(ctx)
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(plans) < 4 {
		t.Fatalf("expected at least 4 seeded plans, got %d", len(plans))
	}

	// 2. Get Single Plan
	p, err := store.GetPlan(ctx, "ultimate_monthly")
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if p.Tier != domain.TierUltimate {
		t.Fatalf("expected tier ultimate, got %s", p.Tier)
	}

	// 3. Upsert Subscription for Clinic A
	now := time.Now().UTC()
	subA := domain.ClinicSubscription{
		ID:                 id.New(),
		WorkspaceID:        clinicA,
		PlanID:             "ultimate_monthly",
		Status:             domain.StatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		PlansCreatedCount:  0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	ctxA := ctxkey.WithClinicID(ctx, clinicA.String())
	ctxB := ctxkey.WithClinicID(ctx, clinicB.String())

	if err := store.UpsertClinicSubscription(ctxA, subA); err != nil {
		t.Fatalf("UpsertClinicSubscription failed: %v", err)
	}

	// 4. Clinic A can fetch its subscription
	gotA, err := store.GetClinicSubscription(ctxA, clinicA)
	if err != nil {
		t.Fatalf("GetClinicSubscription for A failed: %v", err)
	}
	if gotA.PlanID != "ultimate_monthly" {
		t.Fatalf("expected plan_id ultimate_monthly, got %s", gotA.PlanID)
	}
	if gotA.Plan.Tier != domain.TierUltimate {
		t.Fatalf("expected hydrated tier ultimate, got %s", gotA.Plan.Tier)
	}

	// 5. IDOR test: Clinic B CANNOT fetch Clinic A's subscription
	_, err = store.GetClinicSubscription(ctxB, clinicA)
	if !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for Clinic B querying Clinic A subscription due to RLS, got: %v", err)
	}

	// 6. Increment plan count
	if err := store.IncrementPlanCount(ctxA, clinicA); err != nil {
		t.Fatalf("IncrementPlanCount failed: %v", err)
	}
	gotAAfter, err := store.GetClinicSubscription(ctxA, clinicA)
	if err != nil {
		t.Fatalf("GetClinicSubscription failed: %v", err)
	}
	if gotAAfter.PlansCreatedCount != 1 {
		t.Fatalf("expected count 1, got %d", gotAAfter.PlansCreatedCount)
	}

	// 7. Demo Requests
	demo := domain.DemoRequest{
		ID:          id.New(),
		WebsiteURL:  "https://dentsmile.de",
		ClinicName:  "DentSmile Berlin",
		ContactName: "Dr. Klaus",
		Email:       "klaus@dentsmile.de",
		Phone:       "+491512345678",
		Notes:       "Interested in Ultimate plan with 3D player",
		Status:      "pending",
		CreatedAt:   now,
	}
	if err := store.CreateDemoRequest(ctx, demo); err != nil {
		t.Fatalf("CreateDemoRequest failed: %v", err)
	}

	demos, err := store.ListDemoRequests(ctx)
	if err != nil {
		t.Fatalf("ListDemoRequests failed: %v", err)
	}
	if len(demos) == 0 {
		t.Fatal("expected at least 1 demo request")
	}
}
