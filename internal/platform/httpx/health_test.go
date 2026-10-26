package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubReady struct {
	name string
	err  error
}

func (s stubReady) Name() string { return s.name }

func (s stubReady) Ready(context.Context) error { return s.err }

func TestHealthzOK(t *testing.T) {
	rec := httptest.NewRecorder()
	handleHealthz(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestReadyzFailsWhenDependencyDown(t *testing.T) {
	h := handleReadyz([]Readiness{
		stubReady{name: "postgres"},
		stubReady{name: "redis", err: errors.New("down")},
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", rec.Code)
	}
	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "not_ready" {
		t.Fatalf("status %s", body.Status)
	}
}

func TestReadyzOK(t *testing.T) {
	h := handleReadyz([]Readiness{stubReady{name: "postgres"}, stubReady{name: "redis"}})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}
