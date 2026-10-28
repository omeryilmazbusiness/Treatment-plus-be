package http_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	identitydomain "github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	mediaapp "github.com/treatment-plus/treatment-plus-be/internal/media/app"
	mediahttp "github.com/treatment-plus/treatment-plus-be/internal/media/http"
	mediamem "github.com/treatment-plus/treatment-plus-be/internal/media/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

var jpegSOF = []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x43, 0x00}

func TestMediaUploadAndReject(t *testing.T) {
	clk := clock.System{}
	jwtAuth, err := authn.NewJWT(config.Auth{JWTAccessSecret: "test-jwt-secret-min-32-bytes-long!", AccessTTL: 15 * time.Minute}, clk)
	if err != nil {
		t.Fatal(err)
	}
	svc := mediaapp.NewService(mediamem.New())
	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		mediahttp.NewHandler(svc, jwtAuth).Register(v1)
	})
	ws, user := id.New(), id.New()
	token, _, err := jwtAuth.Issue(user, ws, identitydomain.RoleClinic, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", "logo.jpg")
	_, _ = fw.Write(append(jpegSOF, bytes.Repeat([]byte{0}, 64)...))
	_ = w.WriteField("kind", "logo")
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/v1/media", &body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["key"] == nil || out["kind"] != "logo" {
		t.Fatalf("%v", out)
	}

	bad := httptest.NewRequest(http.MethodPost, "/v1/media", bytes.NewReader([]byte("nope")))
	bad.Header.Set("Authorization", "Bearer "+token)
	bad.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	r.ServeHTTP(badRec, bad)
	if badRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", badRec.Code)
	}
}
