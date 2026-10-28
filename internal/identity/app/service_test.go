package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/identity/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) { return "hash:" + password, nil }

func (fakeHasher) Verify(hash, password string) error {
	if hash != "hash:"+password {
		return errx.ErrUnauthenticated
	}
	return nil
}

type fakeAccess struct{}

func (fakeAccess) Issue(userID, workspaceID id.ID, role domain.Role, _ time.Time) (string, time.Duration, error) {
	return "access-" + workspaceID.String() + "-" + string(role), 15 * time.Minute, nil
}

func newService(t *testing.T) (*app.Service, *mem.Store) {
	t.Helper()
	store := mem.New()
	clk := clock.Fixed{T: time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)}
	svc, err := app.NewService(store, store, store, store, fakeHasher{}, fakeAccess{}, clk, 7*24*time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	return svc, store
}

func TestRegisterClinicAndPersonal(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	clinic, err := svc.Register(ctx, app.RegisterInput{
		Name: "Klinik A", Email: "a@clinic.com", Password: "secret-1a", Role: "clinic",
		Clinic: app.RequiredClinic("+90 555 111 2233"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if clinic.Account.User.Role != domain.RoleClinic || clinic.Account.Workspace.Kind != domain.RoleClinic {
		t.Fatalf("%+v", clinic.Account)
	}
	if !strings.Contains(clinic.Tokens.Access, "clinic") {
		t.Fatalf("token %s", clinic.Tokens.Access)
	}

	personal, err := svc.Register(ctx, app.RegisterInput{
		Name: "Dr Ada", Email: "ada@solo.com", Password: "secret-1a", Role: "personal",
	})
	if err != nil {
		t.Fatal(err)
	}
	if personal.Account.User.Role != domain.RolePersonal {
		t.Fatalf("role %s", personal.Account.User.Role)
	}

	if _, err := svc.Register(ctx, app.RegisterInput{
		Name: "Same Mail", Email: "a@clinic.com", Password: "secret-1a", Role: "personal",
	}); !errors.Is(err, errx.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, app.RegisterInput{
		Name: "Klinik A", Email: "a@clinic.com", Password: "secret-1a", Role: "clinic",
		Clinic: app.RequiredClinic("+90 555 111 2233"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(ctx, reg.Tokens.Refresh); err != nil {
		t.Fatal(err)
	}
	login, err := svc.Login(ctx, app.LoginInput{Email: "a@clinic.com", Password: "secret-1a"})
	if err != nil {
		t.Fatal(err)
	}
	if login.Account.User.Role != domain.RoleClinic {
		t.Fatal(login.Account.User.Role)
	}
	refreshed, err := svc.Refresh(ctx, login.Tokens.Refresh)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(ctx, login.Tokens.Refresh); !errors.Is(err, errx.ErrUnauthenticated) {
		t.Fatalf("reuse %v", err)
	}
	if _, err := svc.Refresh(ctx, refreshed.Tokens.Refresh); !errors.Is(err, errx.ErrUnauthenticated) {
		t.Fatalf("family %v", err)
	}
}

func TestGetWorkspaceIDOR(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	a, err := svc.Register(ctx, app.RegisterInput{Name: "Klinik A", Email: "a@clinic.com", Password: "secret-1a", Role: "clinic", Clinic: app.RequiredClinic("+90 555 111 2233")})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Register(ctx, app.RegisterInput{Name: "Ada", Email: "b@solo.com", Password: "secret-1b", Role: "personal"})
	if err != nil {
		t.Fatal(err)
	}
	aCtx := ctxkey.WithClinicID(ctx, a.Account.Workspace.ID.String())
	if _, err := svc.GetWorkspace(aCtx, b.Account.Workspace.ID); !errors.Is(err, errx.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
	me, err := svc.Me(aCtx)
	if err != nil {
		t.Fatal(err)
	}
	if me.Workspace.ID != a.Account.Workspace.ID {
		t.Fatal(me.Workspace.ID)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc, _ := newService(t)
	if _, err := svc.Register(context.Background(), app.RegisterInput{Name: "A", Email: "bad", Password: "x", Role: "clinic"}); !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
	if _, err := svc.Register(context.Background(), app.RegisterInput{Name: "Klinik A", Email: "a@clinic.com", Password: "secret-1a", Role: "admin"}); !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
	if _, err := svc.Register(context.Background(), app.RegisterInput{Name: "Klinik A", Email: "c@clinic.com", Password: "secret-1a", Role: "clinic"}); !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("clinic without phone: %v", err)
	}
}
