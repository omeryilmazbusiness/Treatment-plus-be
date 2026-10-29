package domain_test

import (
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

func TestLink_IsActive(t *testing.T) {
	now := time.Now().UTC()
	l := domain.Link{
		ID:        id.New(),
		ExpiresAt: now.Add(1 * time.Hour),
	}
	if !l.IsActive(now) {
		t.Fatal("expected link to be active")
	}
	if l.IsExpired(now) {
		t.Fatal("expected link not to be expired")
	}

	// Revoked
	rev := now.Add(-5 * time.Minute)
	l.RevokedAt = &rev
	if l.IsActive(now) {
		t.Fatal("expected revoked link not to be active")
	}
	if !l.IsRevoked() {
		t.Fatal("expected link to be revoked")
	}

	// Expired
	l.RevokedAt = nil
	l.ExpiresAt = now.Add(-10 * time.Minute)
	if l.IsActive(now) {
		t.Fatal("expected expired link not to be active")
	}
	if !l.IsExpired(now) {
		t.Fatal("expected link to be expired")
	}
}

func TestHashToken(t *testing.T) {
	h1 := domain.HashToken("test-token-123")
	h2 := domain.HashToken("test-token-123")
	h3 := domain.HashToken("different-token")

	if string(h1) != string(h2) {
		t.Fatal("same tokens must produce same hash")
	}
	if string(h1) == string(h3) {
		t.Fatal("different tokens must produce different hashes")
	}
}
