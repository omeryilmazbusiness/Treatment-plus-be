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
	planpg "github.com/treatment-plus/treatment-plus-be/internal/plan/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
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
	return planapp.ClinicSettingsDTO{Name: "Test Klinik", Phone: "555"}, nil
}

func setupPlanTestServer(t *testing.T) (*chi.Mux, *authn.JWT, *planapp.Service, id.ID, id.ID) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.NewPool(t.Context(), cfg.Database)
	if err != nil {
		t.Skipf("postgres pool not available: %v", err)
	}
	t.Cleanup(pool.Close)

	clk := clock.System{}
	jwtAuth, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-secret-min-32-bytes-long!",
		AccessTTL:       15 * time.Minute,
	}, clk)
	if err != nil {
		t.Fatal(err)
	}

	enc := crypto.NewAESGCM("secret-test-key-32b-length-ok!!!")
	store := planpg.New(pool)
	svc := planapp.NewService(store, store, &dummyCatalog{}, &dummySettings{}, enc)
	h := planhttp.NewHandler(svc, jwtAuth)

	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		h.Register(v1)
	})

	wsID := id.New()
	userID := id.New()
	return r, jwtAuth, svc, wsID, userID
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
	r, jwtAuth, _, wsID, userID := setupPlanTestServer(t)
	token := issueToken(t, jwtAuth, userID, wsID)

	// 1. Create Patient
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

	// 2. Create Plan
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

	// 3. Put Visits
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

	// 4. Put Items
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

	// 5. Ready
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
