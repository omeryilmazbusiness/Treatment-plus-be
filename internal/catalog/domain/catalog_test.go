package domain_test

import (
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/catalog/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

func TestDiagnosis_Validate(t *testing.T) {
	d := domain.Diagnosis{Code: "D01", Name: "Çürük"}
	if err := d.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dEmpty := domain.Diagnosis{Code: "", Name: "Çürük"}
	if err := dEmpty.Validate(); err == nil {
		t.Fatalf("expected error for empty code")
	}
}

func TestTreatment_Validate(t *testing.T) {
	tr := domain.Treatment{Code: "T01", Name: "Kompozit Dolgu"}
	if err := tr.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trEmpty := domain.Treatment{Code: "T01", Name: ""}
	if err := trEmpty.Validate(); err == nil {
		t.Fatalf("expected error for empty name")
	}
}

func TestPriceList_Validate(t *testing.T) {
	pl := domain.PriceList{Name: "2026 Standart", Currency: money.TRY}
	if err := pl.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plBadCurr := domain.PriceList{Name: "2026 Standart", Currency: "ABC"}
	if err := plBadCurr.Validate(); err == nil {
		t.Fatalf("expected error for invalid currency")
	}
}
