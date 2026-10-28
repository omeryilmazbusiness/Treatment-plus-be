package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	identitydomain "github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	planapp "github.com/treatment-plus/treatment-plus-be/internal/plan/app"
	plandomain "github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	planhttp "github.com/treatment-plus/treatment-plus-be/internal/plan/http"
	planmem "github.com/treatment-plus/treatment-plus-be/internal/plan/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/crypto"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type dummyCatalog struct{}

func (d *dummyCatalog) GetTreatment(ctx context.Context, wsID id.ID, treatID id.ID) (string, string, int64, error) {
	return "DUMMY", "Dummy Tedavi", 10000, nil
}

type dummySettings struct{}

func (d *dummySettings) GetClinicSettings(ctx context.Context, wsID id.ID) (planapp.ClinicSettingsDTO, error) {
	return planapp.ClinicSettingsDTO{Name: "Test Klinik", Phone: "555", Locale: "tr", Currency: "TRY", Palette: "navy"}, nil
}

func setupPlanTestServer(t *testing.T) (*chi.Mux, *authn.JWT, id.ID, id.ID) {
	t.Helper()
	clk := clock.System{}
	jwtAuth, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-secret-min-32-bytes-long!",
		AccessTTL:       15 * time.Minute,
	}, clk)
	if err != nil {
		t.Fatal(err)
	}

	store := planmem.New()
	svc := planapp.NewService(store, store, &dummyCatalog{}, &dummySettings{}, crypto.NewAESGCM("secret-test-key-32b-length-ok!!!"))
	h := planhttp.NewHandler(svc, jwtAuth)

	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		h.Register(v1)
	})

	return r, jwtAuth, id.New(), id.New()
}

func issueToken(t *testing.T, jwtAuth *authn.JWT, userID, wsID id.ID) string {
	t.Helper()
	token, _, err := jwtAuth.Issue(userID, wsID, identitydomain.RoleClinic, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestPlanHTTP_CRUD(t *testing.T) {
	r, jwtAuth, wsID, userID := setupPlanTestServer(t)
	token := issueToken(t, jwtAuth, userID, wsID)

	patReqBody := map[string]any{"name": "Fatma Hanım", "contact": "555-1122"}
	bPat, _ := json.Marshal(patReqBody)
	reqPat := httptest.NewRequest(http.MethodPost, "/v1/patients", bytes.NewReader(bPat))
	reqPat.Header.Set("Authorization", "Bearer "+token)
	reqPat.Header.Set("Content-Type", "application/json")
	recPat := httptest.NewRecorder()
	r.ServeHTTP(recPat, reqPat)

	if recPat.Code != http.StatusCreated {
		t.Fatalf("POST /patients failed: %d, body: %s", recPat.Code, recPat.Body.String())
	}
	var createdPat plandomain.Patient
	_ = json.Unmarshal(recPat.Body.Bytes(), &createdPat)

	planReqBody := map[string]any{
		"patient_id": createdPat.ID.String(),
		"title":      "Fatma Hanım Plan",
		"currency":   "TRY",
		"diagnoses":  []string{"Gingivitis"},
	}
	bPlan, _ := json.Marshal(planReqBody)
	reqPlan := httptest.NewRequest(http.MethodPost, "/v1/plans", bytes.NewReader(bPlan))
	reqPlan.Header.Set("Authorization", "Bearer "+token)
	reqPlan.Header.Set("Content-Type", "application/json")
	recPlan := httptest.NewRecorder()
	r.ServeHTTP(recPlan, reqPlan)

	if recPlan.Code != http.StatusCreated {
		t.Fatalf("POST /plans failed: %d, body: %s", recPlan.Code, recPlan.Body.String())
	}
	var createdPlan plandomain.Plan
	_ = json.Unmarshal(recPlan.Body.Bytes(), &createdPlan)

	vBody := map[string]any{
		"visits": []map[string]any{
			{"title": "1. Seans"},
			{"title": "2. Seans"},
		},
	}
	bV, _ := json.Marshal(vBody)
	reqV := httptest.NewRequest(http.MethodPut, "/v1/plans/"+createdPlan.ID.String()+"/visits", bytes.NewReader(bV))
	reqV.Header.Set("Authorization", "Bearer "+token)
	reqV.Header.Set("Content-Type", "application/json")
	recV := httptest.NewRecorder()
	r.ServeHTTP(recV, reqV)

	if recV.Code != http.StatusOK {
		t.Fatalf("PUT /plans/:id/visits failed: %d", recV.Code)
	}

	iBody := map[string]any{
		"items": []map[string]any{
			{
				"treatment_name": "Diş Temizliği",
				"quantity":       1,
				"unit_price":     120000,
				"is_package":     false,
			},
			{
				"treatment_name": "Paket Hijyen Kiti",
				"quantity":       1,
				"unit_price":     20000,
				"is_package":     true,
			},
		},
	}
	bI, _ := json.Marshal(iBody)
	reqI := httptest.NewRequest(http.MethodPut, "/v1/plans/"+createdPlan.ID.String()+"/items", bytes.NewReader(bI))
	reqI.Header.Set("Authorization", "Bearer "+token)
	reqI.Header.Set("Content-Type", "application/json")
	recI := httptest.NewRecorder()
	r.ServeHTTP(recI, reqI)

	if recI.Code != http.StatusOK {
		t.Fatalf("PUT /plans/:id/items failed: %d", recI.Code)
	}
	var updatedWithItems plandomain.Plan
	_ = json.Unmarshal(recI.Body.Bytes(), &updatedWithItems)
	if updatedWithItems.TotalsSubtotal != 120000 {
		t.Fatalf("expected subtotal 120000, got %d", updatedWithItems.TotalsSubtotal)
	}

	reqReady := httptest.NewRequest(http.MethodPost, "/v1/plans/"+createdPlan.ID.String()+"/ready", nil)
	reqReady.Header.Set("Authorization", "Bearer "+token)
	recReady := httptest.NewRecorder()
	r.ServeHTTP(recReady, reqReady)

	if recReady.Code != http.StatusOK {
		t.Fatalf("POST /plans/:id/ready failed: %d", recReady.Code)
	}
	var readyPlan plandomain.Plan
	_ = json.Unmarshal(recReady.Body.Bytes(), &readyPlan)
	if readyPlan.Status != plandomain.StatusReady {
		t.Fatalf("expected status ready, got %s", readyPlan.Status)
	}
}

func TestPlanHTTP_ClassicFormAndSubmit(t *testing.T) {
	r, jwtAuth, wsID, userID := setupPlanTestServer(t)
	token := issueToken(t, jwtAuth, userID, wsID)

	formReq := httptest.NewRequest(http.MethodGet, "/v1/plans/classic/form", nil)
	formReq.Header.Set("Authorization", "Bearer "+token)
	formRec := httptest.NewRecorder()
	r.ServeHTTP(formRec, formReq)
	if formRec.Code != http.StatusOK {
		t.Fatalf("classic form %d %s", formRec.Code, formRec.Body.String())
	}
	var form planapp.ClassicForm
	if err := json.Unmarshal(formRec.Body.Bytes(), &form); err != nil {
		t.Fatal(err)
	}
	if form.Product != "classic" || form.Clinic.Name != "Test Klinik" {
		t.Fatalf("form %+v", form)
	}
	if len(form.Sections) != 5 || form.Sections[0].ID != "presentation" || form.Sections[4].ID != "discounts" {
		t.Fatalf("section order %+v", form.Sections)
	}

	body := map[string]any{
		"palette":   "navy",
		"locale":    "tr",
		"patient":   map[string]any{"name": "Fatma Hanım", "contact": "555-1122"},
		"title":     "Fatma Hanım Plan",
		"situation": map[string]any{"general_diagnoses": []string{"Gingivitis"}, "tooth_findings": []map[string]any{{"tooth": "16", "status": "missing"}}},
		"visits":    []map[string]any{{"title": "1. Seans", "healing_note": "2 hafta", "chart_teeth": []string{"16"}}},
		"items": []map[string]any{
			{"treatment_name": "Diş Temizliği", "quantity": 1, "unit_price": 120000, "teeth": []string{"16"}},
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/plans/classic", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("classic submit %d %s", rec.Code, rec.Body.String())
	}
	var plan plandomain.Plan
	if err := json.Unmarshal(rec.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Currency != "TRY" || plan.TotalsSubtotal != 120000 || plan.Palette != "navy" || plan.Locale != "tr" {
		t.Fatalf("plan %+v", plan)
	}
	if len(plan.Situation.ToothFindings) != 1 || plan.Situation.ToothFindings[0].Tooth != "16" {
		t.Fatalf("situation %+v", plan.Situation)
	}
	if plan.Patient == nil || plan.Patient.Name != "Fatma Hanım" {
		t.Fatalf("patient %+v", plan.Patient)
	}

	missing := map[string]any{
		"patient": map[string]any{"name": "Ali"},
		"items":   []map[string]any{{"treatment_name": "Kontrol", "quantity": 1, "unit_price": 1000}},
	}
	rawMissing, _ := json.Marshal(missing)
	missReq := httptest.NewRequest(http.MethodPost, "/v1/plans/classic", bytes.NewReader(rawMissing))
	missReq.Header.Set("Authorization", "Bearer "+token)
	missReq.Header.Set("Content-Type", "application/json")
	missRec := httptest.NewRecorder()
	r.ServeHTTP(missRec, missReq)
	if missRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("classic without palette %d %s", missRec.Code, missRec.Body.String())
	}
}
