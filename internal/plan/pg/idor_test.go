package pg_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	catalogapp "github.com/treatment-plus/treatment-plus-be/internal/catalog/app"
	catalogpg "github.com/treatment-plus/treatment-plus-be/internal/catalog/pg"
	clinicapp "github.com/treatment-plus/treatment-plus-be/internal/clinic/app"
	clinicdomain "github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	clinicpg "github.com/treatment-plus/treatment-plus-be/internal/clinic/pg"
	identityapp "github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	identitypg "github.com/treatment-plus/treatment-plus-be/internal/identity/pg"
	planapp "github.com/treatment-plus/treatment-plus-be/internal/plan/app"
	plandomain "github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	planpg "github.com/treatment-plus/treatment-plus-be/internal/plan/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/crypto"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

func init() {
	wd, _ := os.Getwd()
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			_ = godotenv.Load(candidate)
			break
		}
		dir = filepath.Dir(dir)
	}
}

type testCatalogAdapter struct {
	svc *catalogapp.Service
}

func (c *testCatalogAdapter) GetTreatment(ctx context.Context, wsID id.ID, treatID id.ID) (string, string, int64, error) {
	price, t, err := c.svc.GetTreatmentPrice(ctx, wsID, treatID)
	if err != nil {
		return "", "", 0, err
	}
	return t.Code, t.Name, price, nil
}

type testSettingsAdapter struct {
	svc *clinicapp.Service
}

func (c *testSettingsAdapter) GetClinicSettings(ctx context.Context, wsID id.ID) (planapp.ClinicSettingsDTO, error) {
	st, err := c.svc.GetSettings(ctx, wsID)
	if err != nil {
		return planapp.ClinicSettingsDTO{}, err
	}
	return planapp.ClinicSettingsDTO{
		Name:    st.Name,
		Phone:   st.Phone,
		Palette: string(st.Palette),
		LogoKey: st.LogoKey,
		Intro:   st.Intro,
		Legal:   st.Legal,
	}, nil
}

func TestPlan_FullLifecycleAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		t.Skipf("postgres pool not available: %v", err)
	}
	t.Cleanup(pool.Close)

	clk := clock.System{}
	tokens, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-access-secret-32bytes-ok!",
		AccessTTL:       time.Minute,
	}, clk)
	if err != nil {
		t.Fatal(err)
	}

	idStore := identitypg.New(pool)
	idSvc, err := identityapp.NewService(idStore, idStore, idStore, idStore, authn.FastArgon2id(), tokens, clk, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	suffix := id.New().String()
	regA, err := idSvc.Register(ctx, identityapp.RegisterInput{
		Name: "Klinik Alpha", Email: "alpha-" + suffix + "@test.com", Password: "password123", Role: "clinic",
	})
	if err != nil {
		t.Fatal(err)
	}
	regB, err := idSvc.Register(ctx, identityapp.RegisterInput{
		Name: "Klinik Beta", Email: "beta-" + suffix + "@test.com", Password: "password123", Role: "clinic",
	})
	if err != nil {
		t.Fatal(err)
	}

	wsAID := regA.Account.Workspace.ID
	wsBID := regB.Account.Workspace.ID
	ctxA := ctxkey.WithClinicID(ctx, wsAID.String())
	ctxB := ctxkey.WithClinicID(ctx, wsBID.String())

	// Services setup
	clinicStore := clinicpg.New(pool)
	clinicSvc := clinicapp.NewService(clinicStore)
	_ = clinicStore.SaveSettings(ctxA, clinicdomain.Settings{
		WorkspaceID: wsAID,
		Name:        "Alpha Diş Kliniği",
		Phone:       "+90 532 000 1122",
		Palette:     clinicdomain.PaletteNavy,
	})

	catalogStore := catalogpg.New(pool)
	catalogSvc := catalogapp.NewService(catalogStore)

	encrypter := crypto.NewAESGCM("secret-test-key-32b-length-ok!!!")
	planStore := planpg.New(pool)
	planSvc := planapp.NewService(
		planStore,
		planStore,
		&testCatalogAdapter{svc: catalogSvc},
		&testSettingsAdapter{svc: clinicSvc},
		encrypter,
	)

	// 1. Workspace A creates catalog (Diagnoses, Treatments, Price List)
	diag1, err := catalogSvc.CreateDiagnosis(ctxA, wsAID, "K02.1", "Dentin Çürüğü")
	if err != nil {
		t.Fatalf("create diagnosis: %v", err)
	}
	tr1, err := catalogSvc.CreateTreatment(ctxA, wsAID, "TR-01", "Kompozit Dolgu")
	if err != nil {
		t.Fatalf("create treatment: %v", err)
	}
	tr2, err := catalogSvc.CreateTreatment(ctxA, wsAID, "TR-02", "Zirkonyum Kron")
	if err != nil {
		t.Fatalf("create treatment: %v", err)
	}

	pl, err := catalogSvc.CreatePriceList(ctxA, wsAID, "2026 Standart", money.TRY, true)
	if err != nil {
		t.Fatalf("create price list: %v", err)
	}
	_, err = catalogSvc.SetPriceItems(ctxA, wsAID, pl.ID, []catalogapp.SetPriceItemInput{
		{TreatmentID: tr1.ID, Amount: 150000}, // 1,500.00 TL in kuruş
		{TreatmentID: tr2.ID, Amount: 400000}, // 4,000.00 TL in kuruş
	})
	if err != nil {
		t.Fatalf("set price items: %v", err)
	}

	// 2. Workspace A creates patient (name, contact encrypted)
	patA, err := planSvc.CreatePatient(ctxA, wsAID, "Mehmet Öz", "+90 555 987 6543")
	if err != nil {
		t.Fatalf("create patient: %v", err)
	}
	if patA.Contact != "+90 555 987 6543" {
		t.Errorf("expected decrypted contact in domain, got: %s", patA.Contact)
	}

	// 3. Workspace A creates a classic plan
	planA, err := planSvc.CreatePlan(ctxA, wsAID, planapp.CreatePlanInput{
		PatientID: patA.ID,
		Title:     "Mehmet Öz Tedavi Planı",
		Currency:  "TRY",
		Diagnoses: []string{diag1.Name},
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	if planA.Kind != plandomain.KindClassic {
		t.Fatalf("expected kind classic, got %s", planA.Kind)
	}
	if planA.Status != plandomain.StatusDraft {
		t.Fatalf("expected status draft, got %s", planA.Status)
	}

	// 4. Set 2 Visits
	v1ID := id.New().String()
	v2ID := id.New().String()
	planA, err = planSvc.PutVisits(ctxA, wsAID, planA.ID, []planapp.PutVisitInput{
		{ID: &v1ID, OrderNum: 1, Title: "1. Seans: Dolgular ve Hazırlık"},
		{ID: &v2ID, OrderNum: 2, Title: "2. Seans: Zirkonyum Simantasyon"},
	})
	if err != nil {
		t.Fatalf("put visits: %v", err)
	}
	if len(planA.Visits) != 2 {
		t.Fatalf("expected 2 visits, got %d", len(planA.Visits))
	}

	// 5. Add Items:
	// Item 1: 2x Kompozit Dolgu @ 1,500 TL = 3,000 TL (300,000 kuruş)
	// Item 2: 1x Zirkonyum Kron @ 4,000 TL = 4,000 TL (400,000 kuruş)
	// Item 3: Paket içi röntgen/kontrol @ 500 TL (50,000 kuruş), is_package=true -> contributes 0!
	tr1IDStr := tr1.ID.String()
	tr2IDStr := tr2.ID.String()
	tooth1 := "16"
	tooth2 := "21"
	planA, err = planSvc.PutItems(ctxA, wsAID, planA.ID, []planapp.PutItemInput{
		{
			VisitID:     &v1ID,
			TreatmentID: &tr1IDStr,
			ToothNumber: &tooth1,
			Quantity:    2,
			UnitPrice:   nil, // auto from catalog: 150000
			IsPackage:   false,
		},
		{
			VisitID:     &v2ID,
			TreatmentID: &tr2IDStr,
			ToothNumber: &tooth2,
			Quantity:    1,
			UnitPrice:   nil, // auto from catalog: 400000
			IsPackage:   false,
		},
		{
			VisitID:       &v1ID,
			TreatmentName: "Panoramik Röntgen (Paket İçi)",
			Quantity:      1,
			UnitPrice:     &[]int64{50000}[0],
			IsPackage:     true, // Paket içi satır toplamı şişirmez!
		},
	})
	if err != nil {
		t.Fatalf("put items: %v", err)
	}

	// Subtotal check: 300,000 + 400,000 = 700,000 kuruş (package item was 0)
	if planA.TotalsSubtotal != 700000 {
		t.Fatalf("expected subtotal 700000, got %d", planA.TotalsSubtotal)
	}

	// 6. Add Discount: 50,000 kuruş (500 TL)
	planA, err = planSvc.PutDiscounts(ctxA, wsAID, planA.ID, []planapp.PutDiscountInput{
		{Label: "Nakit İndirimi", Amount: 50000},
	})
	if err != nil {
		t.Fatalf("put discounts: %v", err)
	}

	// Grand total check: 700,000 - 50,000 = 650,000 kuruş
	if planA.TotalsGrand != 650000 {
		t.Fatalf("expected grand total 650000, got %d", planA.TotalsGrand)
	}

	// 7. Re-open / reload plan: must be identical
	reloaded, err := planSvc.GetPlan(ctxA, wsAID, planA.ID)
	if err != nil {
		t.Fatalf("reload plan: %v", err)
	}
	if len(reloaded.Visits) != 2 || len(reloaded.Items) != 3 || len(reloaded.Discounts) != 1 {
		t.Fatalf("reloaded plan structure mismatch: %+v", reloaded)
	}
	if reloaded.TotalsGrand != 650000 {
		t.Fatalf("reloaded grand total mismatch: %d", reloaded.TotalsGrand)
	}

	// 8. Lock snapshot: POST ready
	readyPlan, err := planSvc.SetReady(ctxA, wsAID, planA.ID)
	if err != nil {
		t.Fatalf("set ready: %v", err)
	}
	if readyPlan.Status != plandomain.StatusReady {
		t.Fatalf("expected status ready, got %s", readyPlan.Status)
	}
	if readyPlan.SnapshotJSON == nil {
		t.Fatalf("snapshot_json must not be nil after ready")
	}

	// 9. Mutating a ready plan must fail
	_, err = planSvc.PutItems(ctxA, wsAID, planA.ID, []planapp.PutItemInput{
		{TreatmentName: "Dolgu", Quantity: 1},
	})
	if err == nil {
		t.Fatalf("modifying ready plan must be rejected")
	}

	// 10. Catalog price update does NOT mutate the locked ready snapshot
	_, err = catalogSvc.SetPriceItems(ctxA, wsAID, pl.ID, []catalogapp.SetPriceItemInput{
		{TreatmentID: tr1.ID, Amount: 999999}, // Price increased
	})
	if err != nil {
		t.Fatalf("update catalog price: %v", err)
	}

	// Check reloaded ready plan: totals and snapshot are unchanged!
	frozenPlan, err := planSvc.GetPlan(ctxA, wsAID, planA.ID)
	if err != nil {
		t.Fatalf("get frozen plan: %v", err)
	}
	if frozenPlan.TotalsGrand != 650000 {
		t.Fatalf("ready plan grand total was modified by catalog change! Got %d", frozenPlan.TotalsGrand)
	}

	// 11. Multi-tenant isolation (Workspace B cannot see or touch Workspace A's plan)
	plansB, err := planSvc.ListPlans(ctxB, wsBID)
	if err != nil {
		t.Fatalf("workspace B list plans failed: %v", err)
	}
	if len(plansB) != 0 {
		t.Fatalf("workspace B must see 0 plans, got %d", len(plansB))
	}

	_, err = planSvc.GetPlan(ctxB, wsBID, planA.ID)
	if err == nil || (!errors.Is(err, errx.ErrNotFound) && !errors.Is(err, pgx.ErrNoRows)) {
		t.Fatalf("workspace B getting workspace A plan must return 404 / ErrNotFound, got: %v", err)
	}

	// Workspace B trying to mutate workspace A plan
	_, err = planSvc.PutVisits(ctxB, wsBID, planA.ID, []planapp.PutVisitInput{
		{Title: "Hacked Visit"},
	})
	if err == nil {
		t.Fatalf("workspace B must not be able to put visits on workspace A plan")
	}
}
