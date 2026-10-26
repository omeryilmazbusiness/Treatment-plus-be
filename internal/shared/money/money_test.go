package money

import "testing"

func TestAddSameCurrency(t *testing.T) {
	a, _ := New(1999, TRY)
	b, _ := New(1, TRY)
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Amount != 2000 {
		t.Fatalf("got %d", sum.Amount)
	}
}

func TestAddRejectsMismatch(t *testing.T) {
	a, _ := New(100, TRY)
	b, _ := New(100, EUR)
	if _, err := a.Add(b); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestRejectsUnknownCurrency(t *testing.T) {
	if _, err := New(1, "BTC"); err == nil {
		t.Fatal("expected error")
	}
}
