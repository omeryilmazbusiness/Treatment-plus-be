package domain_test

import (
	"strings"
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

func TestClassicFormSectionsOrder(t *testing.T) {
	sections := domain.ClassicFormSections()
	if len(sections) != 5 {
		t.Fatalf("len %d", len(sections))
	}
	want := []string{"presentation", "patient", "situation", "visits", "discounts"}
	for i, id := range want {
		if sections[i].ID != id || sections[i].Order != i+1 {
			t.Fatalf("section %d = %+v", i, sections[i])
		}
	}
	if sections[0].Fields[0].Name != "palette" || sections[0].Fields[1].Name != "locale" {
		t.Fatalf("presentation fields %+v", sections[0].Fields)
	}
	for _, s := range sections {
		for _, f := range s.Fields {
			if strings.HasPrefix(f.Name, "clinic.") || f.Name == "logo_key" {
				t.Fatalf("clinic brand leaked into plan form: %s", f.Name)
			}
		}
	}
}

func TestCalculateTotals_PackageDoesNotAdd(t *testing.T) {
	items := []domain.Item{
		{
			Quantity:  2,
			UnitPrice: 100000, // 1000.00 TL in kuruş
			IsPackage: false,
		},
		{
			Quantity:  1,
			UnitPrice: 500000, // 5000.00 TL
			IsPackage: true,   // Package item -> should add 0
		},
	}
	discounts := []domain.Discount{
		{
			Label:  "Nakit İndirimi",
			Amount: 20000, // 200.00 TL
		},
	}

	totals, err := domain.CalculateTotals(items, discounts, money.TRY)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Subtotal: 2 * 100000 = 200000 (package ignored)
	if totals.Subtotal != 200000 {
		t.Errorf("expected subtotal 200000, got %d", totals.Subtotal)
	}
	// Discount: 20000
	if totals.Discount != 20000 {
		t.Errorf("expected discount 20000, got %d", totals.Discount)
	}
	// Grand: 200000 - 20000 = 180000
	if totals.Grand != 180000 {
		t.Errorf("expected grand 180000, got %d", totals.Grand)
	}
}

func TestCalculateTotals_DiscountExceedsSubtotal(t *testing.T) {
	items := []domain.Item{
		{Quantity: 1, UnitPrice: 50000, IsPackage: false},
	}
	discounts := []domain.Discount{
		{Amount: 100000},
	}
	totals, err := domain.CalculateTotals(items, discounts, money.TRY)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if totals.Grand != 0 {
		t.Errorf("expected grand total not to go below 0, got %d", totals.Grand)
	}
}

func TestItem_LineTotal(t *testing.T) {
	it := domain.Item{Quantity: 3, UnitPrice: 15000, IsPackage: false}
	if it.LineTotal() != 45000 {
		t.Errorf("expected 45000, got %d", it.LineTotal())
	}
	itPkg := domain.Item{Quantity: 3, UnitPrice: 15000, IsPackage: true}
	if itPkg.LineTotal() != 0 {
		t.Errorf("expected 0 for package item, got %d", itPkg.LineTotal())
	}
}

func TestItemTeethAndSituation(t *testing.T) {
	first, all := domain.ItemTeeth(nil, []string{"16", "16", "17"})
	if first == nil || *first != "16" || len(all) != 2 {
		t.Fatalf("%v %v", first, all)
	}
	sit := domain.MergeSituation([]string{"Gingivitis"}, domain.Situation{
		ToothFindings: []domain.ToothFinding{{Tooth: "16", Status: "missing"}},
	})
	if sit.Diagnoses()[0] != "Gingivitis" || sit.ToothFindings[0].Status != domain.ToothMissing {
		t.Fatalf("%+v", sit)
	}
}

func TestPatient_Validate(t *testing.T) {
	p := domain.Patient{ID: id.New(), Name: "Ahmet Yılmaz"}
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pEmpty := domain.Patient{ID: id.New(), Name: "   "}
	if err := pEmpty.Validate(); err == nil {
		t.Fatalf("expected error for empty name")
	}
}
