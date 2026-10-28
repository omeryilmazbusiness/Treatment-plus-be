// Package pgtest applies migrations and opens the app-role pool for integration tests.
package pgtest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
)

func Open(t *testing.T) *pgxpool.Pool {
	t.Helper()
	loadDotenv()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateUp(cfg.Database.URL); err != nil {
		t.Skipf("postgres/migrate not available: %v", err)
	}
	pool, err := postgres.NewPool(t.Context(), cfg.Database)
	if err != nil {
		t.Skipf("app role not available: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func MigrateUp(dsn string) error {
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

func loadDotenv() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			_ = godotenv.Load(candidate)
			return
		}
		dir = filepath.Dir(dir)
	}
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
