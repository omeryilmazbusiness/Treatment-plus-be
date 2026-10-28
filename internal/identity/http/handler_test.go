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
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
	Workspace struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"workspace"`
	Tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	} `json:"tokens"`
}

type errBody struct {
	Error struct {
		Code string `json:"code"`
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
	svc, err := identityapp.NewService(store, store, store, store, authn.FastArgon2id(), tokens, clk, 7*24*time.Hour, nil)
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

	regA := doJSON(t, h, http.MethodPost, "/auth/register", "", map[string]any{
		"name": "Klinik A", "email": "a@clinic.com", "password": "secret-1a", "role": "clinic",
		"clinic": map[string]string{
			"phone": "+90 555 111 2233", "email": "info@a-clinic.com", "address": "Bağdat Cad. No:1",
			"hours": "09:00-18:00", "slogan": "Every Smile Matters", "signer_name": "Dr. Ada",
			"locale": "tr", "currency": "TRY", "palette": "navy",
		},
	})
	if regA.Code != http.StatusCreated {
		t.Fatalf("register A %d %s", regA.Code, regA.Body.String())
	}
	a := decodeAuth(t, regA)
	if a.User.Role != "clinic" || a.Workspace.Kind != "clinic" {
		t.Fatalf("%+v", a)
	}
	if a.Tokens.TokenType != "Bearer" || a.Tokens.ExpiresIn != 900 {
		t.Fatalf("tokens %+v", a.Tokens)
	}
	if regA.Header().Get("Set-Cookie") != "" {
		t.Fatal("api must not write cookies")
	}

	regP := doJSON(t, h, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Dr Ada", "email": "ada@solo.com", "password": "secret-1b", "role": "personal",
	})
	if regP.Code != http.StatusCreated {
		t.Fatalf("personal %d %s", regP.Code, regP.Body.String())
	}
	p := decodeAuth(t, regP)
	if p.User.Role != "personal" {
		t.Fatalf("role %s", p.User.Role)
	}

	cross := doJSON(t, h, http.MethodGet, "/workspaces/"+p.Workspace.ID, a.Tokens.AccessToken, nil)
	if cross.Code != http.StatusForbidden {
		t.Fatalf("idor %d %s", cross.Code, cross.Body.String())
	}

	me := doJSON(t, h, http.MethodGet, "/auth/me", a.Tokens.AccessToken, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me %d %s", me.Code, me.Body.String())
	}

	own := doJSON(t, h, http.MethodGet, "/clinics/"+a.Workspace.ID, a.Tokens.AccessToken, nil)
	if own.Code != http.StatusOK {
		t.Fatalf("own %d %s", own.Code, own.Body.String())
	}

	logout := doJSON(t, h, http.MethodPost, "/auth/logout", "", map[string]string{"refresh_token": a.Tokens.RefreshToken})
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout %d", logout.Code)
	}
	login := doJSON(t, h, http.MethodPost, "/auth/login", "", map[string]string{"email": "a@clinic.com", "password": "secret-1a"})
	if login.Code != http.StatusOK {
		t.Fatalf("login %d", login.Code)
	}
	logged := decodeAuth(t, login)
	ref := doJSON(t, h, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": logged.Tokens.RefreshToken})
	if ref.Code != http.StatusOK {
		t.Fatalf("refresh %d", ref.Code)
	}
	rotated := decodeAuth(t, ref)
	if doJSON(t, h, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": logged.Tokens.RefreshToken}).Code != http.StatusUnauthorized {
		t.Fatal("reuse")
	}
	if doJSON(t, h, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": rotated.Tokens.RefreshToken}).Code != http.StatusUnauthorized {
		t.Fatal("family")
	}
}

func TestRegisterClinicRequiresPhone(t *testing.T) {
	h := newRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Klinik A", "email": "a@clinic.com", "password": "secret-1a", "role": "clinic",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterRejectsUnknownRole(t *testing.T) {
	h := newRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Klinik A", "email": "a@clinic.com", "password": "secret-1a", "role": "admin",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
}

func TestMeRequiresAuth(t *testing.T) {
	h := newRouter(t)
	if doJSON(t, h, http.MethodGet, "/auth/me", "", nil).Code != http.StatusUnauthorized {
		t.Fatal("expected 401")
	}
}
