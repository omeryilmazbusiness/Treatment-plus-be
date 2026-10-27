package domain

import (
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

func TestNormalizeName(t *testing.T) {
	got, err := NormalizeName("  Klinik   A  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Klinik A" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeName("A"); err == nil {
		t.Fatal("expected short name to fail")
	}
}

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  Clinic@Example.COM ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "clinic@example.com" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeEmail("not-an-email"); err == nil {
		t.Fatal("expected invalid email")
	}
	if _, err := NormalizeEmail("Name <clinic@example.com>"); err == nil {
		t.Fatal("expected display-name form to fail")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected short password to fail")
	}
	if err := ValidatePassword("long-enough"); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshSecretRoundTrip(t *testing.T) {
	raw, hash, err := NewRefreshSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) != 32 {
		t.Fatalf("hash len %d", len(hash))
	}
	got := HashRefresh(raw)
	if string(got) != string(hash) {
		t.Fatal("hash mismatch")
	}
}

func TestSessionUsable(t *testing.T) {
	now := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)
	ok := Session{ExpiresAt: now.Add(time.Hour)}
	if err := ok.Usable(now); err != nil {
		t.Fatal(err)
	}

	reusedAt := now.Add(-time.Minute)
	reused := Session{ExpiresAt: now.Add(time.Hour), ReusedAt: &reusedAt}
	if err := reused.Usable(now); err != ErrRefreshReused {
		t.Fatalf("got %v", err)
	}

	revokedAt := now
	revoked := Session{ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt}
	if err := revoked.Usable(now); !errx.Is(err, errx.ErrUnauthenticated) {
		t.Fatalf("got %v", err)
	}

	expired := Session{ID: id.New(), ExpiresAt: now.Add(-time.Second)}
	if err := expired.Usable(now); !errx.Is(err, errx.ErrUnauthenticated) {
		t.Fatalf("got %v", err)
	}
}
