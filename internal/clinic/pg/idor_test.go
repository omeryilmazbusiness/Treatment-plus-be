package pg_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	clinicdomain "github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	clinicpg "github.com/treatment-plus/treatment-plus-be/internal/clinic/pg"
	identityapp "github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	identitypg "github.com/treatment-plus/treatment-plus-be/internal/identity/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
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

func TestIDOR_WorkspaceB_SeesZeroRows_ForWorkspaceA(t *testing.T) {
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

	idStore := identitypg.New(pool)
	tokens, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-access-secret-32bytes-ok!",
		AccessTTL:       time.Minute,
	}, clock.System{})
	if err != nil {
		t.Fatal(err)
	}
	idSvc, err := identityapp.NewService(idStore, idStore, idStore, idStore, authn.FastArgon2id(), tokens, clock.System{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	suffix := id.New().String()
	regA, err := idSvc.Register(ctx, identityapp.RegisterInput{
		Name: "Klinik A", Email: "clinic-a-" + suffix + "@test.com", Password: "password123", Role: "clinic",
	})
	if err != nil {
		t.Fatal(err)
	}
	regB, err := idSvc.Register(ctx, identityapp.RegisterInput{
		Name: "Klinik B", Email: "clinic-b-" + suffix + "@test.com", Password: "password123", Role: "personal",
	})
	if err != nil {
		t.Fatal(err)
	}

	wsAID := regA.Account.Workspace.ID
	wsBID := regB.Account.Workspace.ID

	clinicStore := clinicpg.New(pool)

	// Workspace A saves settings
	ctxA := ctxkey.WithClinicID(ctx, wsAID.String())
	settingsA := clinicdomain.Settings{
		WorkspaceID: wsAID,
		Name:        "Klinik A Brand",
		Locale:      clinicdomain.LocaleTR,
		Currency:    money.TRY,
		Palette:     clinicdomain.PaletteForest,
		Phone:       "+90 555 111 2233",
		MapURL:      "https://maps.example.com/a",
		Intro:       "Klinik A Intro",
		Legal:       "Klinik A Legal",
	}
	if err := clinicStore.SaveSettings(ctxA, settingsA); err != nil {
		t.Fatalf("Workspace A save settings failed: %v", err)
	}

	// Workspace A can read its own settings
	gotA, err := clinicStore.GetSettings(ctxA, wsAID)
	if err != nil {
		t.Fatalf("Workspace A get settings failed: %v", err)
	}
	if gotA.Name != "Klinik A Brand" || gotA.Palette != clinicdomain.PaletteForest {
		t.Fatalf("settings mismatch for A: %+v", gotA)
	}

	// Workspace B tries to read Workspace A's settings (IDOR attack)
	// Under Workspace B's RLS context (app.clinic_id = B), reading A's settings must see 0 rows (ErrNotFound)
	ctxB := ctxkey.WithClinicID(ctx, wsBID.String())
	_, err = clinicStore.GetSettings(ctxB, wsAID)
	if err == nil || (!errors.Is(err, errx.ErrNotFound) && !errors.Is(err, pgx.ErrNoRows)) {
		t.Fatalf("Workspace B MUST see 0 rows / not found for Workspace A, got: %v", err)
	}

	// Workspace B tries to overwrite Workspace A's settings (IDOR write attack)
	settingsAttack := clinicdomain.Settings{
		WorkspaceID: wsAID, // Malicious target
		Name:        "Hijacked by B",
		Locale:      clinicdomain.LocaleEN,
		Currency:    money.USD,
		Palette:     clinicdomain.PaletteRose,
	}
	err = clinicStore.SaveSettings(ctxB, settingsAttack)
	if err == nil {
		t.Fatalf("Workspace B MUST NOT be able to overwrite Workspace A's settings (expected RLS violation / 403)")
	}

	// Verify Workspace A's settings remain completely unchanged
	verifiedA, err := clinicStore.GetSettings(ctxA, wsAID)
	if err != nil {
		t.Fatalf("re-verifying A failed: %v", err)
	}
	if verifiedA.Name != "Klinik A Brand" || verifiedA.Palette != clinicdomain.PaletteForest {
		t.Fatalf("Workspace A settings were tampered: %+v", verifiedA)
	}
}
