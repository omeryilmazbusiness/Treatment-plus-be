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
	outputapp "github.com/treatment-plus/treatment-plus-be/internal/output/app"
	outputdomain "github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	outputhttp "github.com/treatment-plus/treatment-plus-be/internal/output/http"
	outputmem "github.com/treatment-plus/treatment-plus-be/internal/output/mem"
	outputpdf "github.com/treatment-plus/treatment-plus-be/internal/output/pdf"
	planapp "github.com/treatment-plus/treatment-plus-be/internal/plan/app"
	plandomain "github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	planhttp "github.com/treatment-plus/treatment-plus-be/internal/plan/http"
	planmem "github.com/treatment-plus/treatment-plus-be/internal/plan/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/crypto"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type dummyCatalog struct{}

func (d *dummyCatalog) GetTreatment(context.Context, id.ID, id.ID) (string, string, int64, error) {
	return "X", "X", 0, nil
}

type dummySettings struct{}

func (d *dummySettings) GetClinicSettings(context.Context, id.ID) (planapp.ClinicSettingsDTO, error) {
	return planapp.ClinicSettingsDTO{
		Name: "Wise Dental", Phone: "555", Email: "info@wise.test", Address: "Cad 1", Hours: "09-18",
		Slogan: "Every Smile Matters", SignerName: "Dr Ada", Locale: "tr", Currency: "TRY", Palette: "navy",
	}, nil
}

type planViewAdapter struct{ svc *planapp.Service }

func (a planViewAdapter) GetView(ctx context.Context, clinicID, planID id.ID) (outputdomain.PlanView, error) {
	p, err := a.svc.GetPlan(ctx, clinicID, planID)
	if err != nil {
		return outputdomain.PlanView{}, err
	}
	if !p.IsReadyOrPresented() || p.SnapshotJSON == nil {
		return outputdomain.PlanView{}, errx.New("plan_not_ready", "plan must be ready before PDF generate", errx.ErrInvalid)
	}
	if raw, ok := p.SnapshotJSON.(json.RawMessage); ok {
		return outputdomain.PlanView{PlanID: planID, ClinicID: clinicID, Payload: raw}, nil
	}
	payload, err := json.Marshal(p.SnapshotJSON)
	if err != nil {
		return outputdomain.PlanView{}, err
	}
	return outputdomain.PlanView{PlanID: planID, ClinicID: clinicID, Payload: payload}, nil
}

func TestPlanPDF_GenerateDownloadAndDraftRejected(t *testing.T) {
	clk := clock.System{}
	jwtAuth, err := authn.NewJWT(config.Auth{JWTAccessSecret: "test-jwt-secret-min-32-bytes-long!", AccessTTL: 15 * time.Minute}, clk)
	if err != nil {
		t.Fatal(err)
	}
	store := planmem.New()
	planSvc := planapp.NewService(store, store, &dummyCatalog{}, &dummySettings{}, crypto.NewAESGCM("secret-test-key-32b-length-ok!!!"))
	outSvc := outputapp.NewService(planViewAdapter{svc: planSvc}, outputpdf.NewRenderer(), nil, outputmem.New(), outputpdf.AwardJPEG())

	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		planhttp.NewHandler(planSvc, jwtAuth).Register(v1)
		outputhttp.NewHandler(outSvc, jwtAuth).Register(v1)
	})
	ws, user := id.New(), id.New()
	token, _, err := jwtAuth.Issue(user, ws, identitydomain.RoleClinic, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	auth := func(req *http.Request) *http.Request {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	classic := map[string]any{
		"palette": "navy",
		"locale":  "tr",
		"patient": map[string]any{"name": "Fatma HanÄ±m"},
		"title":   "Fatma HanÄ±m Plan",
		"visits":  []map[string]any{{"title": "1. Seans"}},
		"items":   []map[string]any{{"treatment_name": "Ä°mplant", "quantity": 1, "unit_price": 120000, "teeth": []string{"16"}}},
	}
	raw, _ := json.Marshal(classic)
	create := httptest.NewRecorder()
	r.ServeHTTP(create, auth(httptest.NewRequest(http.MethodPost, "/v1/plans/classic", bytes.NewReader(raw))))
	if create.Code != http.StatusCreated {
		t.Fatalf("classic %d %s", create.Code, create.Body.String())
	}
	var plan plandomain.Plan
	_ = json.Unmarshal(create.Body.Bytes(), &plan)

	draftPDF := httptest.NewRecorder()
	r.ServeHTTP(draftPDF, auth(httptest.NewRequest(http.MethodPost, "/v1/plans/"+plan.ID.String()+"/pdf", nil)))
	if draftPDF.Code != http.StatusUnprocessableEntity {
		t.Fatalf("draft pdf %d %s", draftPDF.Code, draftPDF.Body.String())
	}

	ready := httptest.NewRecorder()
	r.ServeHTTP(ready, auth(httptest.NewRequest(http.MethodPost, "/v1/plans/"+plan.ID.String()+"/ready", nil)))
	if ready.Code != http.StatusOK {
		t.Fatalf("ready %d %s", ready.Code, ready.Body.String())
	}

	gen := httptest.NewRecorder()
	r.ServeHTTP(gen, auth(httptest.NewRequest(http.MethodPost, "/v1/plans/"+plan.ID.String()+"/pdf", nil)))
	if gen.Code != http.StatusCreated {
		t.Fatalf("generate %d %s", gen.Code, gen.Body.String())
	}
	var meta map[string]any
	_ = json.Unmarshal(gen.Body.Bytes(), &meta)
	if meta["page_count"] != float64(24) {
		t.Fatalf("meta %+v", meta)
	}

	dl := httptest.NewRecorder()
	r.ServeHTTP(dl, auth(httptest.NewRequest(http.MethodGet, "/v1/plans/"+plan.ID.String()+"/pdf", nil)))
	if dl.Code != http.StatusOK || dl.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("download %d %s", dl.Code, dl.Header().Get("Content-Type"))
	}
	if !bytes.HasPrefix(dl.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("body %q", dl.Body.Bytes()[:8])
	}
}
