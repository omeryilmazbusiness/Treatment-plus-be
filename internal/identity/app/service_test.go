package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/app"
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

func (fakeAccess) Issue(clinicID id.ID, _ time.Time) (string, time.Duration, error) {
	return "access-" + clinicID.String(), 15 * time.Minute, nil
}

func newService(t *testing.T) (*app.Service, *mem.Store) {
	t.Helper()
	store := mem.New()
	clk := clock.Fixed{T: time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)}
	svc, err := app.NewService(store, store, store, store, fakeHasher{}, fakeAccess{}, clk, 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return svc, store
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, app.RegisterInput{
		Name: "Klinik A", Email: "a@clinic.com", Password: "secret-1a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reg.Clinic.Email != "a@clinic.com" || !strings.HasPrefix(reg.Tokens.Access, "access-") {
		t.Fatalf("register result %+v", reg)
	}

	_, err = svc.Register(ctx, app.RegisterInput{
		Name: "Klinik A2", Email: "A@clinic.com", Password: "secret-1a",
	})
	if !errors.Is(err, errx.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}

	if err := svc.Logout(ctx, reg.Tokens.Refresh); err != nil {
		t.Fatal(err)
	}

	login, err := svc.Login(ctx, app.LoginInput{Email: "a@clinic.com", Password: "secret-1a"})
	if err != nil {
		t.Fatal(err)
	}

	refreshed, err := svc.Refresh(ctx, login.Tokens.Refresh)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Tokens.Refresh == login.Tokens.Refresh {
		t.Fatal("refresh must rotate")
	}

	if _, err := svc.Refresh(ctx, login.Tokens.Refresh); !errors.Is(err, errx.ErrUnauthenticated) {
		t.Fatalf("reuse must fail, got %v", err)
	}
	if _, err := svc.Refresh(ctx, refreshed.Tokens.Refresh); !errors.Is(err, errx.ErrUnauthenticated) {
		t.Fatalf("family must be revoked after reuse, got %v", err)
	}

	again, err := svc.Login(ctx, app.LoginInput{Email: "a@clinic.com", Password: "secret-1a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(ctx, again.Tokens.Refresh); err != nil {
		t.Fatal(err)
	}
}

func TestLoginRejectsBadPassword(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, app.RegisterInput{
		Name: "Klinik A", Email: "a@clinic.com", Password: "secret-1a",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(ctx, app.LoginInput{Email: "a@clinic.com", Password: "wrong-pass"}); !errors.Is(err, errx.ErrUnauthenticated) {
		t.Fatalf("got %v", err)
	}
	if _, err := svc.Login(ctx, app.LoginInput{Email: "missing@clinic.com", Password: "secret-1a"}); !errors.Is(err, errx.ErrUnauthenticated) {
		t.Fatalf("got %v", err)
	}
}

func TestGetClinicIDOR(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	a, err := svc.Register(ctx, app.RegisterInput{Name: "Klinik A", Email: "a@clinic.com", Password: "secret-1a"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Register(ctx, app.RegisterInput{Name: "Klinik B", Email: "b@clinic.com", Password: "secret-1b"})
	if err != nil {
		t.Fatal(err)
	}

	aCtx := ctxkey.WithClinicID(ctx, a.Clinic.ID.String())
	if _, err := svc.GetClinic(aCtx, b.Clinic.ID); !errors.Is(err, errx.ErrForbidden) {
		t.Fatalf("expected 403 equivalent, got %v", err)
	}
	me, err := svc.Me(aCtx)
	if err != nil {
		t.Fatal(err)
	}
	if me.ID != a.Clinic.ID {
		t.Fatalf("me %s", me.ID)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc, _ := newService(t)
	_, err := svc.Register(context.Background(), app.RegisterInput{Name: "A", Email: "bad", Password: "x"})
	if !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}
