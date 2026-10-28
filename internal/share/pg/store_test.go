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
	planapp "github.com/treatment-plus/treatment-plus-be/internal/plan/app"
	planpg "github.com/treatment-plus/treatment-plus-be/internal/plan/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres/pgtest"
	sharedomain "github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	sharepg "github.com/treatment-plus/treatment-plus-be/internal/share/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/crypto"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type dummyClinicSettingsAdapter struct {
	svc *clinicapp.Service
}

func (d *dummyClinicSettingsAdapter) GetClinicSettings(ctx context.Context, wsID id.ID) (planapp.ClinicSettingsDTO, error) {
	st, err := d.svc.GetSettings(ctx, wsID)
	if err != nil {
		return planapp.ClinicSettingsDTO{}, err
	}
	return planapp.ClinicSettingsDTO{
		Name:     st.Name,
		Phone:    st.Phone,
		Palette:  string(st.Palette),
		LogoKey:  st.LogoKey,
		Intro:    st.Intro,
		Legal:    st.Legal,
		Locale:   string(st.Locale),
		Currency: string(st.Currency),
	}, nil
}

func TestSharePG_LifecycleIDORAndTelemetry(t *testing.T) {
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
		Name: "Clinic Alpha", Email: "alpha-" + suffix + "@test.com", Password: "password123", Role: "clinic",
		Clinic: identityapp.RequiredClinic("+90 555 111 2233"),
	})
	if err != nil {
		t.Fatal(err)
	}
	clinicA := regA.Account.Workspace.ID

	regB, err := idSvc.Register(ctx, identityapp.RegisterInput{
		Name: "Clinic Beta", Email: "beta-" + suffix + "@test.com", Password: "password123", Role: "clinic",
		Clinic: identityapp.RequiredClinic("+90 555 999 8877"),
	})
	if err != nil {
		t.Fatal(err)
	}
	clinicB := regB.Account.Workspace.ID

	// Create plan for Clinic A
	encrypter := crypto.NewAESGCM("local-dev-kms-key-must-be-at-least-32b")
	planStore := planpg.New(pool)
	planSvc := planapp.NewService(planStore, planStore, nil, &dummyClinicSettingsAdapter{svc: clinicSvc}, encrypter)

	ctxA := ctxkey.WithClinicID(ctx, clinicA.String())
	ctxB := ctxkey.WithClinicID(ctx, clinicB.String())

	patA, err := planSvc.CreatePatient(ctxA, clinicA, "Patient A", "+447700900000")
	if err != nil {
		t.Fatal(err)
	}
	planA, err := planSvc.CreatePlan(ctxA, clinicA, planapp.CreatePlanInput{
		PatientID: patA.ID,
		Title:     "Ultimate Makeover",
		Kind:      "ultimate",
		Palette:   "navy",
		Locale:    "en",
	})
	if err != nil {
		t.Fatal(err)
	}

	shareStore := sharepg.New(pool)

	// 1. Create share link for Clinic A
	rawToken := "sample-raw-token-1234567890-test"
	tokenHash := sharedomain.HashToken(rawToken)
	now := time.Now().UTC()
	linkID := id.New()
	link := sharedomain.Link{
		ID:          linkID,
		ClinicID:    clinicA,
		PlanID:      planA.ID,
		TokenHash:   tokenHash,
		TokenPrefix: rawToken[:8],
		ExpiresAt:   now.Add(30 * 24 * time.Hour),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := shareStore.CreateLink(ctxA, link); err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	// 2. IDOR check: Clinic B cannot access Clinic A's link
	_, err = shareStore.GetLink(ctxB, clinicB, linkID)
	if !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for Clinic B accessing Clinic A link, got: %v", err)
	}

	// Clinic A can access its own link
	gotLink, err := shareStore.GetLink(ctxA, clinicA, linkID)
	if err != nil {
		t.Fatalf("GetLink failed for Clinic A: %v", err)
	}
	if gotLink.ID != linkID {
		t.Fatalf("got link id %s, want %s", gotLink.ID, linkID)
	}

	// 3. ResolveToken via SECURITY DEFINER function
	resolved, err := shareStore.ResolveToken(ctx, tokenHash)
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}
	if resolved.ID != linkID {
		t.Fatalf("resolved id %s, want %s", resolved.ID, linkID)
	}
	if resolved.ViewsCount != 1 {
		t.Fatalf("expected views count 1, got %d", resolved.ViewsCount)
	}

	// Second resolve increments views count
	resolved2, err := shareStore.ResolveToken(ctx, tokenHash)
	if err != nil {
		t.Fatalf("ResolveToken second call failed: %v", err)
	}
	if resolved2.ViewsCount != 2 {
		t.Fatalf("expected views count 2, got %d", resolved2.ViewsCount)
	}

	// 4. Record telemetry events
	err = shareStore.RecordEvent(ctx, tokenHash, id.New(), sharedomain.EventOpened, map[string]any{"device": "mobile"})
	if err != nil {
		t.Fatalf("RecordEvent failed: %v", err)
	}
	err = shareStore.RecordEvent(ctx, tokenHash, id.New(), sharedomain.EventVizPlayed, map[string]any{"sec": 10})
	if err != nil {
		t.Fatalf("RecordEvent video failed: %v", err)
	}

	// Clinic A lists events
	eventsA, err := shareStore.ListEvents(ctxA, clinicA, planA.ID)
	if err != nil {
		t.Fatalf("ListEvents failed for Clinic A: %v", err)
	}
	if len(eventsA) < 2 {
		t.Fatalf("expected at least 2 events for Clinic A, got %d", len(eventsA))
	}

	// Clinic B lists events -> sees zero events
	eventsB, err := shareStore.ListEvents(ctxB, clinicB, planA.ID)
	if err != nil {
		t.Fatalf("ListEvents failed for Clinic B: %v", err)
	}
	if len(eventsB) != 0 {
		t.Fatalf("expected 0 events for Clinic B, got %d", len(eventsB))
	}

	// 5. Patient Accepts Plan
	err = shareStore.AcceptPlan(ctx, tokenHash, "Patient A Signature", "127.0.0.1")
	if err != nil {
		t.Fatalf("AcceptPlan failed: %v", err)
	}

	// Check resolved link is now accepted
	resolvedAfterAccept, err := shareStore.ResolveToken(ctx, tokenHash)
	if err != nil {
		t.Fatalf("ResolveToken after accept failed: %v", err)
	}
	if !resolvedAfterAccept.IsAccepted() {
		t.Fatalf("expected plan link to be accepted")
	}
	if resolvedAfterAccept.AcceptedSignature != "Patient A Signature" {
		t.Fatalf("expected signature 'Patient A Signature', got %s", resolvedAfterAccept.AcceptedSignature)
	}

	// 6. Revoke Link
	// Clinic B cannot revoke Clinic A's link
	err = shareStore.RevokeLink(ctxB, clinicB, linkID)
	if !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for Clinic B revoking Clinic A link, got: %v", err)
	}

	// Clinic A revokes its link
	err = shareStore.RevokeLink(ctxA, clinicA, linkID)
	if err != nil {
		t.Fatalf("RevokeLink failed: %v", err)
	}

	// ResolveToken on revoked link must return error
	_, err = shareStore.ResolveToken(ctx, tokenHash)
	if err == nil {
		t.Fatalf("expected error resolving revoked link, got nil")
	}
	if !errors.Is(err, errx.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for revoked link, got: %v", err)
	}
}
