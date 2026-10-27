package authn

import (
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

func TestArgon2idRoundTrip(t *testing.T) {
	h := FastArgon2id()
	encoded, err := h.Hash("correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Verify(encoded, "correct-horse"); err != nil {
		t.Fatal(err)
	}
	if err := h.Verify(encoded, "wrong-password"); err != errx.ErrUnauthenticated {
		t.Fatalf("got %v", err)
	}
}

func TestJWTIssueAndParse(t *testing.T) {
	now := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)
	j, err := NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-access-secret-32bytes-ok!",
		AccessTTL:       15 * time.Minute,
	}, clock.Fixed{T: now})
	if err != nil {
		t.Fatal(err)
	}
	userID, workspaceID := id.New(), id.New()
	raw, ttl, err := j.Issue(userID, workspaceID, "clinic", now)
	if err != nil {
		t.Fatal(err)
	}
	if ttl != 15*time.Minute {
		t.Fatalf("ttl %s", ttl)
	}
	got, err := j.ParseAccess(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != userID || got.WorkspaceID != workspaceID || string(got.Role) != "clinic" {
		t.Fatalf("got %+v", got)
	}
	if _, err := j.ParseAccess("not-a-jwt"); err != errx.ErrUnauthenticated {
		t.Fatalf("got %v", err)
	}
}

func TestBearer(t *testing.T) {
	if Bearer("Bearer abc") != "abc" {
		t.Fatal("bearer")
	}
	if Bearer("token abc") != "" {
		t.Fatal("expected empty")
	}
}
