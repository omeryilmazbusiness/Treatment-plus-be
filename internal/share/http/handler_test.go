package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	identitydomain "github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/share/app"
	sharehttp "github.com/treatment-plus/treatment-plus-be/internal/share/http"
	"github.com/treatment-plus/treatment-plus-be/internal/share/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
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

func setupShareServer(t *testing.T) (*chi.Mux, *authn.JWT, *app.Service, id.ID, id.ID, id.ID) {
	t.Helper()
	clk := clock.System{}
	jwtAuth, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-secret-min-32-bytes-long!",
		AccessTTL:       15 * time.Minute,
	}, clk)
	if err != nil {
		t.Fatal(err)
	}

	clinicID := id.New()
	userID := id.New()
	planID := id.New()

	plans := &fakePlanReader{
		plan: app.SharedPlanData{
			ID:          planID,
			ClinicID:    clinicID,
			PatientName: "John Smith",
			Title:       "Gülüş Tasarımı",
			Kind:        "ultimate",
			Status:      "ready",
			Currency:    "TRY",
			TotalsGrand: 2500000,
			Visits: []app.SharedVisit{
				{
					ID:         "v-1",
					OrderNum:   1,
					Title:      "1. Seans: Cerrahi",
					PhaseLabel: "Faz 1",
					Items: []app.SharedItem{
						{TreatmentName: "İmplant Yerleşimi", Quantity: 2, UnitPrice: 1000000, Teeth: []string{"16", "26"}},
					},
				},
			},
		},
	}
	clinics := &fakeClinicReader{
		clinic: app.SharedClinicData{
			Name:        "Royal Dental Clinic",
			Slogan:      "Premium Care",
			Phone:       "+90 555 987 6543",
			Doctor:      app.SharedDoctor{Name: "Dr. Sarah Jenkins", Title: "Başhekim"},
			CTA:         app.SharedCTAConfig{WhatsAppNumber: "+905559876543", EnableAcceptance: true},
			WelcomeBody: "Kliniğimize hoş geldiniz.",
		},
	}
	viz := &fakeVizReader{}
	pdf := &fakePDFReader{}
	store := mem.New()

	svc := app.NewService(store, plans, clinics, viz, pdf, "http://localhost:8080")
	handler := sharehttp.NewHandler(svc, jwtAuth)

	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		handler.Register(v1)
	})
	// Root /p/{token} alias
	r.Get("/p/{token}", func(w http.ResponseWriter, req *http.Request) {
		req.URL.Path = "/v1/p/" + chi.URLParam(req, "token")
		r.ServeHTTP(w, req)
	})

	return r, jwtAuth, svc, clinicID, userID, planID
}

func TestShareHTTP_EndToEnd(t *testing.T) {
	r, jwtAuth, _, clinicID, userID, planID := setupShareServer(t)

	// Auth token for clinic
	token, _, err := jwtAuth.Issue(userID, clinicID, identitydomain.RoleClinic, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// 1. Create Share Link: POST /v1/plans/{id}/share
	createBody := bytes.NewBufferString(`{"ttl_hours": 72}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/plans/"+planID.String()+"/share", createBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/plans/{id}/share status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}

	var createRes struct {
		Token     string `json:"token"`
		PublicURL string `json:"public_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &createRes); err != nil {
		t.Fatal(err)
	}
	if createRes.Token == "" {
		t.Fatal("empty token in response")
	}

	// 2. Public GET /v1/public/plans/{token}
	pubReq := httptest.NewRequest(http.MethodGet, "/v1/public/plans/"+createRes.Token, nil)
	pubRec := httptest.NewRecorder()
	r.ServeHTTP(pubRec, pubReq)

	if pubRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/public/plans/{token} status = %d, want 200; body: %s", pubRec.Code, pubRec.Body.String())
	}

	var planDTO app.UltimatePlanDTO
	if err := json.Unmarshal(pubRec.Body.Bytes(), &planDTO); err != nil {
		t.Fatal(err)
	}
	if planDTO.Plan.PatientName != "John Smith" {
		t.Fatalf("expected patient John Smith, got %s", planDTO.Plan.PatientName)
	}
	if planDTO.Clinic.Name != "Royal Dental Clinic" {
		t.Fatalf("expected clinic Royal Dental Clinic, got %s", planDTO.Clinic.Name)
	}
	if len(planDTO.Plan.Visits) != 1 {
		t.Fatalf("expected 1 visit, got %d", len(planDTO.Plan.Visits))
	}

	// 3. Public POST /v1/public/plans/{token}/events
	eventBody := bytes.NewBufferString(`{"kind": "opened", "metadata": {"device": "mobile"}}`)
	evReq := httptest.NewRequest(http.MethodPost, "/v1/public/plans/"+createRes.Token+"/events", eventBody)
	evReq.Header.Set("Content-Type", "application/json")
	evRec := httptest.NewRecorder()
	r.ServeHTTP(evRec, evReq)

	if evRec.Code != http.StatusOK {
		t.Fatalf("POST /v1/public/plans/{token}/events status = %d, want 200", evRec.Code)
	}

	// 4. Public POST /v1/public/plans/{token}/accept
	acceptBody := bytes.NewBufferString(`{"signature": "John Smith"}`)
	accReq := httptest.NewRequest(http.MethodPost, "/v1/public/plans/"+createRes.Token+"/accept", acceptBody)
	accReq.Header.Set("Content-Type", "application/json")
	accRec := httptest.NewRecorder()
	r.ServeHTTP(accRec, accReq)

	if accRec.Code != http.StatusOK {
		t.Fatalf("POST /v1/public/plans/{token}/accept status = %d, want 200", accRec.Code)
	}

	// 5. Public GET /v1/public/plans/{token}/viz
	vizReq := httptest.NewRequest(http.MethodGet, "/v1/public/plans/"+createRes.Token+"/viz", nil)
	vizRec := httptest.NewRecorder()
	r.ServeHTTP(vizRec, vizReq)

	if vizRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/public/plans/{token}/viz status = %d, want 200", vizRec.Code)
	}

	// 6. Public GET /v1/public/plans/{token}/pdf
	pdfReq := httptest.NewRequest(http.MethodGet, "/v1/public/plans/"+createRes.Token+"/pdf", nil)
	pdfRec := httptest.NewRecorder()
	r.ServeHTTP(pdfRec, pdfReq)

	if pdfRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/public/plans/{token}/pdf status = %d, want 200", pdfRec.Code)
	}
	if pdfRec.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("expected Content-Type application/pdf, got %s", pdfRec.Header().Get("Content-Type"))
	}

	// 7. Public GET /v1/p/{token} (HTML Preview)
	htmlReq := httptest.NewRequest(http.MethodGet, "/v1/p/"+createRes.Token, nil)
	htmlRec := httptest.NewRecorder()
	r.ServeHTTP(htmlRec, htmlReq)

	if htmlRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/p/{token} status = %d, want 200", htmlRec.Code)
	}
	if !strings.Contains(htmlRec.Body.String(), "John Smith") {
		t.Fatalf("expected HTML to contain patient name John Smith")
	}

	// 8. Clinic GET /v1/plans/{id}/analytics
	anReq := httptest.NewRequest(http.MethodGet, "/v1/plans/"+planID.String()+"/analytics", nil)
	anReq.Header.Set("Authorization", "Bearer "+token)
	anRec := httptest.NewRecorder()
	r.ServeHTTP(anRec, anReq)

	if anRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/plans/{id}/analytics status = %d, want 200", anRec.Code)
	}

	var analytics app.AnalyticsSummaryDTO
	if err := json.Unmarshal(anRec.Body.Bytes(), &analytics); err != nil {
		t.Fatal(err)
	}
	if !analytics.IsAccepted {
		t.Fatal("expected analytics to report plan accepted")
	}
	if analytics.Signature != "John Smith" {
		t.Fatalf("expected signature John Smith, got %s", analytics.Signature)
	}
}

func TestShareHTTP_ErrorCases(t *testing.T) {
	r, jwtAuth, svc, clinicID, userID, planID := setupShareServer(t)
	token, _, err := jwtAuth.Issue(userID, clinicID, identitydomain.RoleClinic, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// 1. Unauthenticated share creation -> 401
	unauthReq := httptest.NewRequest(http.MethodPost, "/v1/plans/"+planID.String()+"/share", bytes.NewBufferString(`{}`))
	unauthReq.Header.Set("Content-Type", "application/json")
	unauthRec := httptest.NewRecorder()
	r.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", unauthRec.Code)
	}

	// 2. Non-existent token lookup -> 404
	badTokenReq := httptest.NewRequest(http.MethodGet, "/v1/public/plans/non-existent-token-abc", nil)
	badTokenRec := httptest.NewRecorder()
	r.ServeHTTP(badTokenRec, badTokenReq)
	if badTokenRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent token, got %d", badTokenRec.Code)
	}

	// 3. Create a valid link
	createBody := bytes.NewBufferString(`{"ttl_hours": 24}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/plans/"+planID.String()+"/share", createBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var createRes struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &createRes)

	// 4. Accept without signature -> 422
	emptySigReq := httptest.NewRequest(http.MethodPost, "/v1/public/plans/"+createRes.Token+"/accept", bytes.NewBufferString(`{"signature": "   "}`))
	emptySigReq.Header.Set("Content-Type", "application/json")
	emptySigRec := httptest.NewRecorder()
	r.ServeHTTP(emptySigRec, emptySigReq)
	if emptySigRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for empty signature, got %d", emptySigRec.Code)
	}

	// 5. Revoke link then verify access is blocked -> 403
	links, _ := svc.ListShareLinks(context.Background(), clinicID, planID)
	if len(links) > 0 {
		_ = svc.RevokeShareLink(context.Background(), clinicID, links[0].ID)
		revReq := httptest.NewRequest(http.MethodGet, "/v1/public/plans/"+createRes.Token, nil)
		revRec := httptest.NewRecorder()
		r.ServeHTTP(revRec, revReq)
		if revRec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for revoked token, got %d", revRec.Code)
		}
	}
}
