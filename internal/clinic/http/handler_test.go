package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	clinicapp "github.com/treatment-plus/treatment-plus-be/internal/clinic/app"
	clinicdomain "github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	clinichttp "github.com/treatment-plus/treatment-plus-be/internal/clinic/http"
	clinicmem "github.com/treatment-plus/treatment-plus-be/internal/clinic/mem"
	identitydomain "github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

func setupTestServer(t *testing.T) (*chi.Mux, *authn.JWT, *clinicapp.Service, id.ID, id.ID) {
	t.Helper()
	clk := clock.System{}
	jwtAuth, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-secret-min-32-bytes-long!",
		AccessTTL:       15 * time.Minute,
	}, clk)
	if err != nil {
		t.Fatal(err)
	}

	memStore := clinicmem.New()
	svc := clinicapp.NewService(memStore)
	handler := clinichttp.NewHandler(svc, jwtAuth)

	r := chi.NewRouter()
	r.Get("/workspace", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/v1/workspace"
		r.RequestURI = "/v1/workspace"
		handler.Register(r.Context().Value(struct{}{}).(chi.Router))
	})
	// Root /workspace alias
	r.Get("/workspace", func(w http.ResponseWriter, req *http.Request) {
		prods := svc.GetProducts(req.Context())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"products": prods})
	})

	r.Route("/v1", func(v1 chi.Router) {
		handler.Register(v1)
	})

	wsA := id.New()
	userA := id.New()
	return r, jwtAuth, svc, wsA, userA
}

func issueToken(t *testing.T, jwtAuth *authn.JWT, userID, wsID id.ID) string {
	t.Helper()
	token, _, err := jwtAuth.Issue(userID, wsID, identitydomain.RoleClinic, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestGetWorkspaceProducts(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)

	// Test GET /workspace
	req := httptest.NewRequest(http.MethodGet, "/workspace", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /workspace status = %d, want 200", rec.Code)
	}

	var res struct {
		Products []clinicdomain.Product `json:"products"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(res.Products))
	}
	if res.Products[0].ID != "classic" || !res.Products[0].Open {
		t.Fatalf("classic must be open: %+v", res.Products[0])
	}
	if res.Products[1].ID != "ultimate" || res.Products[1].Open {
		t.Fatalf("ultimate must be closed: %+v", res.Products[1])
	}

	// Test GET /v1/workspace
	req2 := httptest.NewRequest(http.MethodGet, "/v1/workspace", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET /v1/workspace status = %d, want 200", rec2.Code)
	}
}

func TestSettings_ValidationFailuresReturn422(t *testing.T) {
	r, jwtAuth, _, wsA, userA := setupTestServer(t)
	token := issueToken(t, jwtAuth, userA, wsA)

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "invalid palette returns 422",
			body: map[string]any{"palette": "emerald"},
		},
		{
			name: "arbitrary hex palette returns 422",
			body: map[string]any{"palette": "#1E293B"},
		},
		{
			name: "invalid locale returns 422",
			body: map[string]any{"locale": "de"},
		},
		{
			name: "invalid currency returns 422",
			body: map[string]any{"currency": "CAD"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPatch, "/v1/clinic/settings", bytes.NewReader(b))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 422 Unprocessable Entity, got %d. Body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSettings_PatchAndGetSameValues(t *testing.T) {
	r, jwtAuth, _, wsA, userA := setupTestServer(t)
	token := issueToken(t, jwtAuth, userA, wsA)

	patchBody := map[string]any{
		"name":      "Örnek Klinik",
		"locale":    "en",
		"currency":  "EUR",
		"palette":   "forest",
		"logo_key":  "https://cdn.example.com/logo.png",
		"phone":     "+90 212 555 0101",
		"map_url":   "https://maps.google.com/?q=acme",
		"intro":     "Uzman kadromuzla yanınızdayız.",
		"legal":     "KVKK kapsamında onaylanmıştır.",
	}
	b, _ := json.Marshal(patchBody)
	patchReq := httptest.NewRequest(http.MethodPatch, "/v1/clinic/settings", bytes.NewReader(b))
	patchReq.Header.Set("Authorization", "Bearer "+token)
	patchReq.Header.Set("Content-Type", "application/json")
	patchRec := httptest.NewRecorder()
	r.ServeHTTP(patchRec, patchReq)

	if patchRec.Code != http.StatusOK {
		t.Fatalf("PATCH expected 200, got %d: %s", patchRec.Code, patchRec.Body.String())
	}

	var patchRes map[string]any
	if err := json.Unmarshal(patchRec.Body.Bytes(), &patchRes); err != nil {
		t.Fatal(err)
	}

	// Now GET the settings
	getReq := httptest.NewRequest(http.MethodGet, "/v1/clinic/settings", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var getRes map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &getRes); err != nil {
		t.Fatal(err)
	}

	// Verify PATCH and GET return identical values
	fields := []string{"name", "locale", "currency", "palette", "logo_key", "phone", "map_url", "intro", "legal"}
	for _, f := range fields {
		if getRes[f] != patchBody[f] {
			t.Errorf("field %s mismatch: got %v, want %v", f, getRes[f], patchBody[f])
		}
		if getRes[f] != patchRes[f] {
			t.Errorf("field %s mismatch between PATCH response and GET: %v vs %v", f, patchRes[f], getRes[f])
		}
	}
	if getRes["workspace_id"] != wsA.String() {
		t.Errorf("workspace_id mismatch: got %v, want %v", getRes["workspace_id"], wsA.String())
	}
}

func TestSettings_BodyWorkspaceIDCannotHijack(t *testing.T) {
	r, jwtAuth, _, wsA, userA := setupTestServer(t)
	token := issueToken(t, jwtAuth, userA, wsA)

	wsB := id.New()
	patchBody := map[string]any{
		"workspace_id": wsB.String(), // Attacker tries to hijack workspace B
		"name":         "Hacked Clinic",
	}
	b, _ := json.Marshal(patchBody)
	req := httptest.NewRequest(http.MethodPatch, "/v1/clinic/settings", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Must be forbidden (403)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when body tries to pass different workspace_id, got %d", rec.Code)
	}
}
