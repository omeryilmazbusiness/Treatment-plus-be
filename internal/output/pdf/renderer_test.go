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
	plandomain "github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
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
			Legal:       "1. Veri Sorumlusu\n[Klinik/Hekim Adı tarafından işlenir.\n2. İşlenme Amacı\nSağlık hizmeti sunumu.",
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
			{VisitTitle: "2. Seans", TreatmentName: "Zirkonyum Kuron", Teeth: []string{"21, 23, 25"}, Quantity: 3, UnitPrice: 50000, LineTotal: 150000},
		},
		Totals:  domain.Totals{Subtotal: 270000, Grand: 270000},
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

func TestClassicRenderer_NoDevLeaksInCopy(t *testing.T) {
	for _, loc := range []string{"tr", "en"} {
		c := domain.CopyFor(loc)
		if strings.Contains(c.UltimateBody, "GET /v1") || strings.Contains(c.UltimateBody, "Cycles") {
			t.Errorf("locale %s UltimateBody contains internal API leak: %s", loc, c.UltimateBody)
		}
		if strings.Contains(c.AwardsDefault, "Treatment Plus standart markasıdır") || strings.Contains(c.AwardsDefault, "standard mark") {
			t.Errorf("locale %s AwardsDefault contains placeholder text: %s", loc, c.AwardsDefault)
		}
		if c.AboutDefault == "" || c.LabDefault == "" || c.HotelDefault == "" {
			t.Errorf("locale %s missing default copy fallbacks", loc)
		}
	}
}

func TestPlanDomain_ItemTeethTokenization(t *testing.T) {
	tooth := "21, 23, 25"
	teeth := []string{"21", "23", "25"}
	first, all := plandomain.ItemTeeth(&tooth, teeth)
	if first == nil || *first != "21" {
		t.Fatalf("expected first tooth 21, got %v", first)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 teeth without duplicates, got %d: %v", len(all), all)
	}
	if all[0] != "21" || all[1] != "23" || all[2] != "25" {
		t.Fatalf("unexpected teeth list: %v", all)
	}
}

func TestClassicRenderer_FallbacksWhenFieldsEmpty(t *testing.T) {
	snap := sampleSnapshot()
	snap.Clinic.Highlights = nil
	snap.Clinic.Lab.Text = ""
	snap.Clinic.Lab.Stats = nil
	snap.Clinic.Hotel.Text = ""
	snap.Clinic.AboutText = ""
	snap.Situation.ToothFindings = nil // All findings empty -> should map from items & defaults

	raw, pages, err := pdf.NewRenderer().Render(context.Background(), snap, app.Images{})
	if err != nil {
		t.Fatal(err)
	}
	if pages != 24 {
		t.Fatalf("expected 24 pages, got %d", pages)
	}
	if len(raw) < 20_000 {
		t.Fatalf("generated PDF too small: %d bytes", len(raw))
	}
}
