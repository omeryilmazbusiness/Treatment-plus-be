package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	identitydomain "github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/app"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/domain"
	vizhttp "github.com/treatment-plus/treatment-plus-be/internal/viz/http"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/player"
)

type stubPlans struct{}

func (stubPlans) GetReadyItems(_ context.Context, _, planID id.ID) (string, string, []app.ItemRef, error) {
	return "tr", planID.String(), []app.ItemRef{
		{AnimationCode: "implant", TreatmentName: "İmplant", Teeth: []string{"16"}},
	}, nil
}

func setup(t *testing.T) (*chi.Mux, *authn.JWT, id.ID, id.ID) {
	t.Helper()
	jwtAuth, err := authn.NewJWT(config.Auth{
		JWTAccessSecret: "test-jwt-secret-min-32-bytes-long!",
		AccessTTL:       15 * time.Minute,
	}, clock.System{})
	if err != nil {
		t.Fatal(err)
	}
	h := vizhttp.NewHandler(app.NewService(stubPlans{}), jwtAuth)
	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) { h.Register(v1) })
	return r, jwtAuth, id.New(), id.New()
}

func TestRecipesAndDemoAndPlayer(t *testing.T) {
	r, _, _, _ := setup(t)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/viz/recipes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("recipes %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Recipes []domain.Recipe `json:"recipes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Recipes) != len(domain.AllCodes()) {
		t.Fatalf("recipes %d", len(body.Recipes))
	}

	demo := httptest.NewRecorder()
	r.ServeHTTP(demo, httptest.NewRequest(http.MethodGet, "/v1/viz/demo?code=implant&teeth=16", nil))
	if demo.Code != http.StatusOK {
		t.Fatalf("demo %d %s", demo.Code, demo.Body.String())
	}
	var pl domain.Playlist
	if err := json.Unmarshal(demo.Body.Bytes(), &pl); err != nil {
		t.Fatal(err)
	}
	if len(pl.Clips) != 1 || pl.Clips[0].Code != domain.Implant || len(pl.Clips[0].Moves) != 3 {
		t.Fatalf("%+v", pl)
	}

	html := httptest.NewRecorder()
	r.ServeHTTP(html, httptest.NewRequest(http.MethodGet, "/v1/viz/player", nil))
	if html.Code != http.StatusOK || !strings.Contains(html.Body.String(), "3D tedavi hamleleri") {
		t.Fatalf("html %d", html.Code)
	}

	js := httptest.NewRecorder()
	r.ServeHTTP(js, httptest.NewRequest(http.MethodGet, "/v1/viz/player.js", nil))
	if js.Code != http.StatusOK || !strings.Contains(js.Body.String(), "GLTFLoader") {
		t.Fatalf("js %d", js.Code)
	}

	cin := httptest.NewRecorder()
	r.ServeHTTP(cin, httptest.NewRequest(http.MethodGet, "/v1/viz/cinematics.js", nil))
	if cin.Code != http.StatusOK || !strings.Contains(cin.Body.String(), "playFissureSealant") || !strings.Contains(cin.Body.String(), "playRootCanal") {
		t.Fatalf("cinematics %d", cin.Code)
	}
}

func TestPlanPlaylistRequiresAuth(t *testing.T) {
	r, jwtAuth, ws, user := setup(t)
	plan := id.New()

	unauth := httptest.NewRecorder()
	r.ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, "/v1/plans/"+plan.String()+"/viz", nil))
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth %d", unauth.Code)
	}

	token, _, err := jwtAuth.Issue(user, ws, identitydomain.RoleClinic, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/plans/"+plan.String()+"/viz", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	authz := httptest.NewRecorder()
	r.ServeHTTP(authz, req)
	if authz.Code != http.StatusOK {
		t.Fatalf("playlist %d %s", authz.Code, authz.Body.String())
	}
}

func TestPlayerEmbedCoversPrimitives(t *testing.T) {
	js := string(player.Cinematics)
	for _, needle := range []string{"step1", "step2", "step3", "step4", "playFissureSealant", "playRootCanal", "getObjectByName", "jawLower_midres"} {
		if !strings.Contains(js, needle) {
			t.Errorf("cinematics.js missing %s", needle)
		}
	}
}
