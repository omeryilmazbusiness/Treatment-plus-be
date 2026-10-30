package mem_test

import (
	"context"
	"errors"
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/plan/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

func TestMemStore_PatientLifecycleAndTenantIsolation(t *testing.T) {
	store := mem.New()
	ws1 := id.New()
	ws2 := id.New()

	ctx1 := ctxkey.WithClinicID(context.Background(), ws1.String())
	ctx2 := ctxkey.WithClinicID(context.Background(), ws2.String())

	pat1 := domain.Patient{
		ID:          id.New(),
		WorkspaceID: ws1,
		Name:        "Hasta 1",
	}

	// 1. Save patient under ws1
	if err := store.SavePatient(ctx1, pat1, "enc:05551112233"); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	// 2. Cross-tenant write forbidden
	pat2 := domain.Patient{
		ID:          id.New(),
		WorkspaceID: ws2,
		Name:        "Hasta 2",
	}
	if err := store.SavePatient(ctx1, pat2, "enc:forbidden"); !errors.Is(err, errx.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cross-tenant save, got %v", err)
	}

	// 3. Get patient by legitimate tenant
	got, err := store.GetPatient(ctx1, ws1, pat1.ID)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if got.Contact != "enc:05551112233" {
		t.Errorf("expected encrypted contact preserved, got %q", got.Contact)
	}

	// 4. Cross-tenant read denied
	if _, err := store.GetPatient(ctx2, ws1, pat1.ID); !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-tenant read, got %v", err)
	}

	// 5. List patients filtered by tenant
	list, err := store.ListPatients(ctx1, ws1)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(list) != 1 || list[0].ID != pat1.ID {
		t.Fatalf("expected 1 patient in list for ws1, got %d", len(list))
	}
}

func TestMemStore_PlanLifecycleAndLockEnforcement(t *testing.T) {
	store := mem.New()
	wsID := id.New()
	ctx := ctxkey.WithClinicID(context.Background(), wsID.String())

	planID := id.New()
	p := domain.Plan{
		ID:          planID,
		WorkspaceID: wsID,
		PatientID:   id.New(),
		Title:       "Test Plan",
		Status:      domain.StatusDraft,
		Currency:    money.TRY,
		Palette:     domain.PaletteNavy,
		Locale:      domain.LocaleTR,
	}

	// 1. Create plan
	if err := store.CreatePlan(ctx, p); err != nil {
		t.Fatalf("unexpected create plan error: %v", err)
	}

	// 2. Update header while in draft
	p.Title = "Updated Title"
	if err := store.UpdatePlanHeader(ctx, p); err != nil {
		t.Fatalf("unexpected update header error: %v", err)
	}
	fetched, _ := store.GetPlan(ctx, wsID, planID)
	if fetched.Title != "Updated Title" {
		t.Errorf("expected updated title, got %q", fetched.Title)
	}

	// 3. Replace visits while in draft
	visits := []domain.Visit{
		{ID: id.New(), PlanID: planID, WorkspaceID: wsID, OrderNum: 1, Title: "Seans 1"},
	}
	if err := store.ReplaceVisits(ctx, wsID, planID, visits); err != nil {
		t.Fatalf("unexpected replace visits error: %v", err)
	}

	// 4. Replace items while in draft
	items := []domain.Item{
		{ID: id.New(), PlanID: planID, WorkspaceID: wsID, TreatmentName: "Dolgu", Quantity: 1, UnitPrice: 50000},
	}
	totals := domain.Totals{Subtotal: 50000, Discount: 0, Grand: 50000}
	if err := store.ReplaceItems(ctx, wsID, planID, items, totals); err != nil {
		t.Fatalf("unexpected replace items error: %v", err)
	}

	// 5. Replace discounts while in draft
	discounts := []domain.Discount{
		{ID: id.New(), PlanID: planID, WorkspaceID: wsID, Label: "İndirim", Amount: 10000},
	}
	totals.Discount = 10000
	totals.Grand = 40000
	if err := store.ReplaceDiscounts(ctx, wsID, planID, discounts, totals); err != nil {
		t.Fatalf("unexpected replace discounts error: %v", err)
	}

	// 6. Lock plan with SetReady
	if err := store.SetReady(ctx, wsID, planID, []byte(`{"ready":true}`)); err != nil {
		t.Fatalf("unexpected set ready error: %v", err)
	}

	lockedPlan, _ := store.GetPlan(ctx, wsID, planID)
	if lockedPlan.Status != domain.StatusReady {
		t.Fatalf("expected status ready, got %s", lockedPlan.Status)
	}

	// 7. Calling SetReady again is idempotent
	if err := store.SetReady(ctx, wsID, planID, []byte(`{"ready":true}`)); err != nil {
		t.Fatalf("idempotent SetReady should succeed, got: %v", err)
	}

	// 8. Modifying locked plan is rejected with ErrNotFound
	if err := store.UpdatePlanHeader(ctx, lockedPlan); !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound modifying locked plan header, got %v", err)
	}
	if err := store.ReplaceVisits(ctx, wsID, planID, visits); !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound replacing visits on locked plan, got %v", err)
	}
	if err := store.ReplaceItems(ctx, wsID, planID, items, totals); !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound replacing items on locked plan, got %v", err)
	}
	if err := store.ReplaceDiscounts(ctx, wsID, planID, discounts, totals); !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound replacing discounts on locked plan, got %v", err)
	}
}

