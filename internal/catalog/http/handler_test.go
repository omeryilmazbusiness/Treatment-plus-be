package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	catalogapp "github.com/treatment-plus/treatment-plus-be/internal/catalog/app"
	catalogdomain "github.com/treatment-plus/treatment-plus-be/internal/catalog/domain"
	cataloghttp "github.com/treatment-plus/treatment-plus-be/internal/catalog/http"
	catalogmem "github.com/treatment-plus/treatment-plus-be/internal/catalog/mem"
	identitydomain "github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

func setupCatalogTestServer(t *testing.T) (*chi.Mux, *authn.JWT, id.ID, id.ID) {
	t.Helper()
	clk := clock.System{}
	jwtAuth, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-secret-min-32-bytes-long!",
		AccessTTL:       15 * time.Minute,
	}, clk)
	if err != nil {
		t.Fatal(err)
	}

	svc := catalogapp.NewService(catalogmem.New())
	h := cataloghttp.NewHandler(svc, jwtAuth)

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

func TestCatalogHTTP_DiagnosesAndTreatments(t *testing.T) {
	r, jwtAuth, wsID, userID := setupCatalogTestServer(t)
	token := issueToken(t, jwtAuth, userID, wsID)

	diagBody := map[string]any{"code": "D-" + wsID.String()[:4], "name": "Periodontitis"}
	b, _ := json.Marshal(diagBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/catalog/diagnoses", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /catalog/diagnoses failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	var createdDiag catalogdomain.Diagnosis
	if err := json.Unmarshal(rec.Body.Bytes(), &createdDiag); err != nil {
		t.Fatal(err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/catalog/diagnoses", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("GET /catalog/diagnoses failed: %d", listRec.Code)
	}

	trBody := map[string]any{"code": "T-" + wsID.String()[:4], "name": "Kanal Tedavisi"}
	bTr, _ := json.Marshal(trBody)
	trReq := httptest.NewRequest(http.MethodPost, "/v1/catalog/treatments", bytes.NewReader(bTr))
	trReq.Header.Set("Authorization", "Bearer "+token)
	trReq.Header.Set("Content-Type", "application/json")
	trRec := httptest.NewRecorder()
	r.ServeHTTP(trRec, trReq)

	if trRec.Code != http.StatusCreated {
		t.Fatalf("POST /catalog/treatments failed: %d, body: %s", trRec.Code, trRec.Body.String())
	}
}

func TestCatalogHTTP_RequiresAuth(t *testing.T) {
	r, _, _, _ := setupCatalogTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/catalog/diagnoses", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET status = %d, want 401", rec.Code)
	}
}
