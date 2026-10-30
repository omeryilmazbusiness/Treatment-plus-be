package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	identitydomain "github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/app"
	subhttp "github.com/treatment-plus/treatment-plus-be/internal/subscription/http"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/mem"
)

func setupSubscriptionServer(t *testing.T) (*chi.Mux, *authn.JWT, *app.Service, id.ID, id.ID) {
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

	store := mem.New()
	svc := app.NewService(store, clk)
	handler := subhttp.NewHandler(svc, jwtAuth)

	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		handler.Register(v1)
	})

	return r, jwtAuth, svc, clinicID, userID
}

func TestSubscriptionHTTP_EndToEnd(t *testing.T) {
	r, jwtAuth, _, clinicID, userID := setupSubscriptionServer(t)

	token, _, err := jwtAuth.Issue(userID, clinicID, identitydomain.RoleClinic, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// 1. Public GET /v1/public/subscriptions/plans
	req := httptest.NewRequest(http.MethodGet, "/v1/public/subscriptions/plans", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// 2. Public POST /v1/public/demo-requests
	demoBody := bytes.NewBufferString(`{
		"website_url": "https://bright-plans.com",
		"clinic_name": "Bright Dental",
		"contact_name": "Dr. Sarah",
		"email": "sarah@bright-plans.com"
	}`)
	demoReq := httptest.NewRequest(http.MethodPost, "/v1/public/demo-requests", demoBody)
	demoReq.Header.Set("Content-Type", "application/json")
	demoRec := httptest.NewRecorder()
	r.ServeHTTP(demoRec, demoReq)
	if demoRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for demo request, got %d", demoRec.Code)
	}

	// 3. Subscribe to classic_monthly: POST /v1/subscriptions/subscribe
	subBody := bytes.NewBufferString(`{"plan_id": "classic_monthly"}`)
	subReq := httptest.NewRequest(http.MethodPost, "/v1/subscriptions/subscribe", subBody)
	subReq.Header.Set("Authorization", "Bearer "+token)
	subReq.Header.Set("Content-Type", "application/json")
	subRec := httptest.NewRecorder()
	r.ServeHTTP(subRec, subReq)
	if subRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for subscribe, got %d body: %s", subRec.Code, subRec.Body.String())
	}

	// 4. GET /v1/subscriptions/current
	curReq := httptest.NewRequest(http.MethodGet, "/v1/subscriptions/current", nil)
	curReq.Header.Set("Authorization", "Bearer "+token)
	curRec := httptest.NewRecorder()
	r.ServeHTTP(curRec, curReq)
	if curRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for current subscription, got %d", curRec.Code)
	}
	var curRes app.SubscriptionSummaryDTO
	if err := json.Unmarshal(curRec.Body.Bytes(), &curRes); err != nil {
		t.Fatal(err)
	}
	if curRes.Tier != "classic" {
		t.Fatalf("expected tier classic, got %s", curRes.Tier)
	}
	if curRes.AllowsUltimate {
		t.Fatal("classic subscription should not allow ultimate")
	}

	// 5. Upgrade to ultimate_yearly: POST /v1/subscriptions/change-tier
	upgBody := bytes.NewBufferString(`{"plan_id": "ultimate_yearly"}`)
	upgReq := httptest.NewRequest(http.MethodPost, "/v1/subscriptions/change-tier", upgBody)
	upgReq.Header.Set("Authorization", "Bearer "+token)
	upgReq.Header.Set("Content-Type", "application/json")
	upgRec := httptest.NewRecorder()
	r.ServeHTTP(upgRec, upgReq)
	if upgRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for upgrade, got %d", upgRec.Code)
	}
	var upgRes app.SubscriptionSummaryDTO
	if err := json.Unmarshal(upgRec.Body.Bytes(), &upgRes); err != nil {
		t.Fatal(err)
	}
	if upgRes.Tier != "ultimate" {
		t.Fatalf("expected tier ultimate, got %s", upgRes.Tier)
	}
	if !upgRes.AllowsUltimate {
		t.Fatal("ultimate subscription must allow ultimate")
	}

	// 6. Cancel subscription: POST /v1/subscriptions/cancel
	cancelReq := httptest.NewRequest(http.MethodPost, "/v1/subscriptions/cancel", nil)
	cancelReq.Header.Set("Authorization", "Bearer "+token)
	cancelRec := httptest.NewRecorder()
	r.ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for cancel, got %d", cancelRec.Code)
	}
	var cancelRes app.SubscriptionSummaryDTO
	if err := json.Unmarshal(cancelRec.Body.Bytes(), &cancelRes); err != nil {
		t.Fatal(err)
	}
	if !cancelRes.CancelAtPeriodEnd {
		t.Fatal("expected CancelAtPeriodEnd to be true")
	}
}

