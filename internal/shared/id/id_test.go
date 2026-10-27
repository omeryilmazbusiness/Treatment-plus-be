package id

import "testing"

func TestNewIsUniqueAndParseable(t *testing.T) {
	a := New()
	b := New()
	if a == b {
		t.Fatal("expected distinct ids")
	}
	parsed, err := Parse(a.String())
	if err != nil {
		t.Fatal(err)
	}
	if parsed != a {
		t.Fatalf("parse round-trip: got %s want %s", parsed, a)
	}
	if len(a.Bytes()) != 16 {
		t.Fatalf("bytes len %d", len(a.Bytes()))
	}
	if len(a.UUIDString()) != 36 {
		t.Fatalf("uuid string %q", a.UUIDString())
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse("not-a-ulid"); err == nil {
		t.Fatal("expected error")
	}
}
