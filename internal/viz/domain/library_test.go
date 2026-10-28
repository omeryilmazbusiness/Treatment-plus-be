package domain_test

import (
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/viz/domain"
)

func TestLibraryCoversEveryCodeAndMove(t *testing.T) {
	lib := domain.Library()
	if len(lib) != len(domain.AllCodes()) {
		t.Fatalf("recipes %d codes %d", len(lib), len(domain.AllCodes()))
	}
	seen := map[domain.Code]int{}
	for _, r := range lib {
		if len(r.Moves) == 0 {
			t.Fatalf("%s has no moves", r.Code)
		}
		for i, m := range r.Moves {
			if m.N != i+1 || m.DurationMS < 1000 || len(m.Effects) == 0 {
				t.Fatalf("%s move %+v", r.Code, m)
			}
			if m.Effects[0].Op != domain.Cam {
				t.Fatalf("%s move %d must start with camera", r.Code, m.N)
			}
		}
		seen[r.Code]++
	}
	for _, c := range domain.AllCodes() {
		if seen[c] != 1 {
			t.Fatalf("code %s count %d", c, seen[c])
		}
	}
}

func TestParseCodeAliases(t *testing.T) {
	c, err := domain.ParseCode("Zirkonyum Kron")
	if err != nil || c != domain.ZirconiaCrown {
		t.Fatalf("%s %v", c, err)
	}
	if _, err := domain.ParseCode("unknown-tx"); err == nil {
		t.Fatal("expected invalid")
	}
	for _, r := range domain.Library() {
		got, err := domain.ParseCode(r.TitleTR)
		if err != nil || got != r.Code {
			t.Fatalf("title %q -> %s %v", r.TitleTR, got, err)
		}
	}
}
