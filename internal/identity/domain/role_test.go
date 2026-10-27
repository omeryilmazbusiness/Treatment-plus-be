package domain

import (
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

func TestParseRole(t *testing.T) {
	got, err := ParseRole(" Clinic ")
	if err != nil || got != RoleClinic {
		t.Fatalf("got %s %v", got, err)
	}
	got, err = ParseRole("personal")
	if err != nil || got != RolePersonal {
		t.Fatalf("got %s %v", got, err)
	}
	if _, err := ParseRole("admin"); err == nil || !errx.Is(err, errx.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}
