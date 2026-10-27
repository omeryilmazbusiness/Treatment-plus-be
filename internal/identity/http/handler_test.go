package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	identityapp "github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	identityhttp "github.com/treatment-plus/treatment-plus-be/internal/identity/http"
	"github.com/treatment-plus/treatment-plus-be/internal/identity/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
)

type authBody struct {
	Clinic struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"clinic"`
	Tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	} `json:"tokens"`
}

type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newRouter(t *testing.T) http.Handler {
	t.Helper()
	store := mem.New()
	clk := clock.System{}
	tokens, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-access-secret-32bytes-ok!",
		AccessTTL:       15 * time.Minute,
	}, clk)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identityapp.NewService(store, store, store, store, authn.FastArgon2id(), tokens, clk, 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	identityhttp.NewHandler(svc, tokens).Register(r)
	return r
}

func doJSON(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeAuth(t *testing.T, rec *httptest.ResponseRecorder) authBody {
	t.Helper()
	var out authBody
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return out
}

func TestAuthFlowAndIDOR(t *testing.T) {
	h := newRouter(t)

	regA := doJSON(t, h, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Klinik A", "email": "a@clinic.com", "password": "secret-1a",
	})
	if regA.Code != http.StatusCreated {
		t.Fatalf("register A %d %s", regA.Code, regA.Body.String())
	}
	a := decodeAuth(t, regA)
	if a.Tokens.TokenType != "Bearer" || a.Tokens.ExpiresIn != 900 {
		t.Fatalf("tokens %+v", a.Tokens)
	}
	if regA.Header().Get("Set-Cookie") != "" {
		t.Fatal("api must not write cookies")
	}
	if rec := doJSON(t, h, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Klinik A2", "email": "a@clinic.com", "password": "secret-1a",
	}); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate %d", rec.Code)
	}

	regB := doJSON(t, h, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Klinik B", "email": "b@clinic.com", "password": "secret-1b",
	})
	if regB.Code != http.StatusCreated {
		t.Fatalf("register B %d %s", regB.Code, regB.Body.String())
	}
	b := decodeAuth(t, regB)

	cross := doJSON(t, h, http.MethodGet, "/clinics/"+b.Clinic.ID, a.Tokens.AccessToken, nil)
	if cross.Code != http.StatusForbidden {
		t.Fatalf("idor status %d %s", cross.Code, cross.Body.String())
	}
	var e errBody
	if err := json.NewDecoder(cross.Body).Decode(&e); err != nil {
		t.Fatal(err)
	}
	if e.Error.Code != "forbidden" {
		t.Fatalf("idor code %s", e.Error.Code)
	}

	me := doJSON(t, h, http.MethodGet, "/auth/me", a.Tokens.AccessToken, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me %d %s", me.Code, me.Body.String())
	}

	own := doJSON(t, h, http.MethodGet, "/clinics/"+a.Clinic.ID, a.Tokens.AccessToken, nil)
	if own.Code != http.StatusOK {
		t.Fatalf("own clinic %d %s", own.Code, own.Body.String())
	}

	logout := doJSON(t, h, http.MethodPost, "/auth/logout", "", map[string]string{
		"refresh_token": a.Tokens.RefreshToken,
	})
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout %d", logout.Code)
	}

	login := doJSON(t, h, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "a@clinic.com", "password": "secret-1a",
	})
	if login.Code != http.StatusOK {
		t.Fatalf("login %d %s", login.Code, login.Body.String())
	}
	logged := decodeAuth(t, login)

	ref := doJSON(t, h, http.MethodPost, "/auth/refresh", "", map[string]string{
		"refresh_token": logged.Tokens.RefreshToken,
	})
	if ref.Code != http.StatusOK {
		t.Fatalf("refresh %d %s", ref.Code, ref.Body.String())
	}
	rotated := decodeAuth(t, ref)

	reuse := doJSON(t, h, http.MethodPost, "/auth/refresh", "", map[string]string{
		"refresh_token": logged.Tokens.RefreshToken,
	})
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("reuse %d %s", reuse.Code, reuse.Body.String())
	}
	family := doJSON(t, h, http.MethodPost, "/auth/refresh", "", map[string]string{
		"refresh_token": rotated.Tokens.RefreshToken,
	})
	if family.Code != http.StatusUnauthorized {
		t.Fatalf("family after reuse %d %s", family.Code, family.Body.String())
	}
}

func TestRegisterValidationHTTP(t *testing.T) {
	h := newRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "A", "email": "bad", "password": "x",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
}

func TestMeRequiresAuth(t *testing.T) {
	h := newRouter(t)
	rec := doJSON(t, h, http.MethodGet, "/auth/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}
