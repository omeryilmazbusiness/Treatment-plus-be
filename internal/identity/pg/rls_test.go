package pg_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	identityapp "github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	identitypg "github.com/treatment-plus/treatment-plus-be/internal/identity/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres/dbgen"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

func TestRLSZeroRowsForOtherClinic(t *testing.T) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateUp(cfg.Database.URL); err != nil {
		t.Skipf("postgres/migrate not available: %v", err)
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		t.Skipf("app role not available: %v", err)
	}
	t.Cleanup(pool.Close)

	store := identitypg.New(pool)
	tokens, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-access-secret-32bytes-ok!",
		AccessTTL:       time.Minute,
	}, clock.System{})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identityapp.NewService(store, store, store, store, authn.FastArgon2id(), tokens, clock.System{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	suffix := id.New().String()
	a, err := svc.Register(ctx, identityapp.RegisterInput{
		Name: "Klinik A", Email: "a-" + suffix + "@clinic.com", Password: "secret-1a",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Register(ctx, identityapp.RegisterInput{
		Name: "Klinik B", Email: "b-" + suffix + "@clinic.com", Password: "secret-1b",
	})
	if err != nil {
		t.Fatal(err)
	}

	aCtx := ctxkey.WithClinicID(ctx, a.Clinic.ID.String())
	if _, err := store.GetVisible(aCtx, b.Clinic.ID); !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("A must see 0 rows for B, got %v", err)
	}
	if _, err := store.GetVisible(aCtx, a.Clinic.ID); err != nil {
		t.Fatalf("A must see itself: %v", err)
	}

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.clinic_id', $1, true)", a.Clinic.ID.UUIDString()); err != nil {
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
		_, err = q.IdentityVisibleClinicID(ctx, uuidFromID(b.Clinic.ID))
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

func migrateUp(dsn string) error {
	dir, err := findMigrations()
	if err != nil {
		return err
	}
	m, err := migrate.New("file://"+dir, dsn)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func findMigrations() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "db", "migrations")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", errors.New("db/migrations not found")
}
