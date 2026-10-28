package pg_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	identityapp "github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	identitypg "github.com/treatment-plus/treatment-plus-be/internal/identity/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres/dbgen"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres/pgtest"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

func TestRLSZeroRowsForOtherClinic(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.Open(t)

	store := identitypg.New(pool)
	tokens, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-access-secret-32bytes-ok!",
		AccessTTL:       time.Minute,
	}, clock.System{})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identityapp.NewService(store, store, store, store, authn.FastArgon2id(), tokens, clock.System{}, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}

	suffix := id.New().String()
	a, err := svc.Register(ctx, identityapp.RegisterInput{
		Name: "Klinik A", Email: "a-" + suffix + "@clinic.com", Password: "secret-1a", Role: "clinic",
		Clinic: identityapp.RequiredClinic("+90 555 111 2233"),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Register(ctx, identityapp.RegisterInput{
		Name: "Klinik B", Email: "b-" + suffix + "@clinic.com", Password: "secret-1b", Role: "personal",
	})
	if err != nil {
		t.Fatal(err)
	}

	aCtx := ctxkey.WithClinicID(ctx, a.Account.Workspace.ID.String())
	if _, err := store.GetVisible(aCtx, b.Account.Workspace.ID); !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("A must see 0 rows for B, got %v", err)
	}
	if _, err := store.GetVisible(aCtx, a.Account.Workspace.ID); err != nil {
		t.Fatalf("A must see itself: %v", err)
	}

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.clinic_id', $1, true)", a.Account.Workspace.ID.UUIDString()); err != nil {
			return err
		}
		q := dbgen.New(tx)
		n, err := q.IdentityCountVisibleClinics(ctx)
		if err != nil {
			return err
		}
		if n != 1 {
			t.Fatalf("visible clinic count %d", n)
		}
		_, err = q.IdentityVisibleClinicID(ctx, uuidFromID(b.Account.Workspace.ID))
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("B row must be hidden, got %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := dbgen.New(tx)
		n, err := q.IdentityCountVisibleClinics(ctx)
		if err != nil {
			return err
		}
		if n != 0 {
			t.Fatalf("without clinic_id, count must be 0, got %d", n)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func uuidFromID(v id.ID) pgtype.UUID {
	var u pgtype.UUID
	copy(u.Bytes[:], v.Bytes())
	u.Valid = true
	return u
}
