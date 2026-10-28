package pdf_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/output/app"
	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/output/pdf"
)

func sampleSnapshot() domain.Snapshot {
	return domain.Snapshot{
		PlanID:   "plan-1",
		Title:    "Fatma Hanım Plan",
		Kind:     "classic",
		Currency: "TRY",
		Palette:  "navy",
		Locale:   "tr",
		Clinic: domain.Clinic{
			Name: "Wise Dental", Slogan: "Every Smile Matters",
			Phone: "+90 212 000 00 00", Email: "info@wise.test", Website: "https://wise.test",
			Address: "Bağdat Cad. No:1", Hours: "09:00-18:00",
			SignerName: "Dr. Ada", SignerTitle: "Klinik Direktörü",
			WelcomeBody: "Gülüşünüz için teşekkürler.",
			Highlights:  []domain.Stat{{Value: "15+", Label: "Yıl"}},
			Doctor:      domain.Doctor{Name: "Dr. Ada", Title: "Prostodontist", Bio: "İmplant ve protez."},
			Lab:         domain.Lab{Title: "Lab", Text: "Dijital laboratuvar."},
			Awards:      domain.Awards{Text: "Avrupa sağlık ödülü."},
			Hotel:       domain.Hotel{Name: "Partner Hotel", Text: "Klinik protokolü."},
		},
		Patient:   domain.Patient{Name: "Fatma Hanım"},
		Diagnoses: []string{"Gingivitis"},
		Situation: domain.Situation{
			GeneralDiagnoses: []string{"Gingivitis"},
			ToothFindings:    []domain.ToothFinding{{Tooth: "16", Status: "missing"}},
			ChartTeeth:       []string{"16"},
		},
		Visits: []domain.Visit{
			{Title: "1. Seans", HealingNote: "2 hafta", ChartTeeth: []string{"16"}},
			{Title: "2. Seans", PhaseLabel: "Protez"},
		},
		Items: []domain.Item{
			{VisitTitle: "1. Seans", TreatmentName: "İmplant", Teeth: []string{"16"}, Quantity: 1, UnitPrice: 120000, LineTotal: 120000},
		},
		Totals:  domain.Totals{Subtotal: 120000, Grand: 120000},
		ReadyAt: time.Date(2026, 10, 28, 8, 0, 0, 0, time.UTC),
	}
}

func TestClassicRenderer_24PagesAndPDFHeader(t *testing.T) {
	raw, pages, err := pdf.NewRenderer().Render(context.Background(), sampleSnapshot(), app.Images{Award: pdf.AwardJPEG()})
	if err != nil {
		t.Fatal(err)
	}
	if pages != 24 {
		t.Fatalf("pages %d", pages)
	}
	if !bytes.HasPrefix(raw, []byte("%PDF")) {
		t.Fatalf("not a pdf: %q", raw[:8])
	}
	if !strings.Contains(string(raw), "/Type /Page") && !strings.Contains(string(raw), "/Type/Page") {
		t.Fatal("pdf missing page objects")
	}
	if len(raw) < 20_000 {
		t.Fatalf("pdf too small: %d", len(raw))
	}
}

func TestClassicRenderer_EnglishPalette(t *testing.T) {
	snap := sampleSnapshot()
	snap.Locale = "en"
	snap.Palette = "forest"
	raw, pages, err := pdf.NewRenderer().Render(context.Background(), snap, app.Images{Award: pdf.AwardJPEG()})
	if err != nil {
		t.Fatal(err)
	}
	if pages != 24 || !bytes.HasPrefix(raw, []byte("%PDF")) {
		t.Fatalf("pages=%d header=%q", pages, raw[:4])
	}
}
