package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/share/app"
	"github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/share/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type fakePlanReader struct {
	plan app.SharedPlanData
}

func (f *fakePlanReader) GetPlanForShare(ctx context.Context, clinicID, planID id.ID) (app.SharedPlanData, error) {
	return f.plan, nil
}

type fakeClinicReader struct {
	clinic app.SharedClinicData
}

func (f *fakeClinicReader) GetClinicBrand(ctx context.Context, clinicID id.ID) (app.SharedClinicData, error) {
	return f.clinic, nil
}

type fakeVizReader struct{}

func (f *fakeVizReader) GetVizPlaylist(ctx context.Context, clinicID, planID id.ID) (app.SharedVizPlaylist, error) {
	return app.SharedVizPlaylist{
		PlanID: planID.String(),
		Locale: "tr",
		Clips: []app.SharedVizClip{
			{Code: "implant", Title: "Dental Implant", Teeth: []string{"16"}},
		},
	}, nil
}

type fakePDFReader struct{}

func (f *fakePDFReader) GetPlanPDF(ctx context.Context, clinicID, planID id.ID) ([]byte, error) {
	return []byte("%PDF-1.4 test"), nil
}

func TestService_ShareFlow(t *testing.T) {
	ctx := context.Background()
	store := mem.New()
	clinicID := id.New()
	planID := id.New()

	plans := &fakePlanReader{
		plan: app.SharedPlanData{
			ID:          planID,
			ClinicID:    clinicID,
			PatientName: "Jane Doe",
			Title:       "Gülüş Tasarımı",
			Kind:        "ultimate",
			Status:      "ready",
			Currency:    "TRY",
			TotalsGrand: 1500000,
		},
	}
	clinics := &fakeClinicReader{
		clinic: app.SharedClinicData{
			Name:   "Bright Dental",
			Slogan: "The modern treatment plan",
			Phone:  "+90 555 123 4567",
			Doctor: app.SharedDoctor{Name: "Dr. Sarah", Title: "Chief Dentist"},
			CTA:    app.SharedCTAConfig{WhatsAppNumber: "+905551234567", EnableAcceptance: true},
		},
	}
	viz := &fakeVizReader{}
	pdf := &fakePDFReader{}

	svc := app.NewService(store, plans, clinics, viz, pdf, "http://localhost:8080")

	// 1. Create Share Link
	res, err := svc.CreateShareLink(ctx, clinicID, planID, 24*time.Hour)
	if err != nil {
		t.Fatalf("create share link: %v", err)
	}
	if res.Token == "" {
		t.Fatal("token cannot be empty")
	}
	if res.PublicURL != "http://localhost:8080/p/"+res.Token {
		t.Fatalf("unexpected public url: %s", res.PublicURL)
	}

	// 2. List Share Links
	links, err := svc.ListShareLinks(ctx, clinicID, planID)
	if err != nil {
		t.Fatalf("list share links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}

	// 3. Get Ultimate Plan as public patient
	ultimate, err := svc.GetUltimatePlan(ctx, res.Token)
	if err != nil {
		t.Fatalf("get ultimate plan: %v", err)
	}
	if ultimate.Plan.PatientName != "Jane Doe" {
		t.Fatalf("expected patient Jane Doe, got %s", ultimate.Plan.PatientName)
	}
	if ultimate.Clinic.Name != "Bright Dental" {
		t.Fatalf("expected clinic Bright Dental, got %s", ultimate.Clinic.Name)
	}
	if ultimate.Viz == nil || len(ultimate.Viz.Clips) != 1 {
		t.Fatalf("expected viz playlist with 1 clip")
	}

	// 4. Record telemetry event
	err = svc.RecordEvent(ctx, res.Token, domain.EventOpened, map[string]any{"source": "sms"})
	if err != nil {
		t.Fatalf("record event: %v", err)
	}

	// 5. Patient accepts plan
	err = svc.AcceptPlan(ctx, res.Token, "Jane Doe", "127.0.0.1")
	if err != nil {
		t.Fatalf("accept plan: %v", err)
	}

	// 6. Check analytics
	analytics, err := svc.GetAnalytics(ctx, clinicID, planID)
	if err != nil {
		t.Fatalf("get analytics: %v", err)
	}
	if !analytics.IsAccepted {
		t.Fatal("plan should be marked as accepted")
	}
	if analytics.Signature != "Jane Doe" {
		t.Fatalf("expected signature Jane Doe, got %s", analytics.Signature)
	}
	if analytics.EventsCount < 2 {
		t.Fatalf("expected at least 2 events (opened + accepted), got %d", analytics.EventsCount)
	}

	// 7. Revoke Link
	err = svc.RevokeShareLink(ctx, clinicID, res.Link.ID)
	if err != nil {
		t.Fatalf("revoke link: %v", err)
	}

	// Now getting ultimate plan with revoked token should fail
	_, err = svc.GetUltimatePlan(ctx, res.Token)
	if err == nil {
		t.Fatal("expected error for revoked token, got nil")
	}
}
