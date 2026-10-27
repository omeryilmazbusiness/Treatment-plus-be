package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/clinic/app"
	"github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type fakeSettingsStore struct {
	data map[id.ID]domain.Settings
}

func newFakeStore() *fakeSettingsStore {
	return &fakeSettingsStore{data: make(map[id.ID]domain.Settings)}
}

func (f *fakeSettingsStore) GetSettings(_ context.Context, wsID id.ID) (domain.Settings, error) {
	s, ok := f.data[wsID]
	if !ok {
		return domain.Settings{}, errx.ErrNotFound
	}
	return s, nil
}

func (f *fakeSettingsStore) SaveSettings(_ context.Context, s domain.Settings) error {
	f.data[s.WorkspaceID] = s
	return nil
}

func ptr[T any](v T) *T { return &v }

func TestService_GetSettingsDefault(t *testing.T) {
	svc := app.NewService(newFakeStore())
	wsID := id.New()
	s, err := svc.GetSettings(context.Background(), wsID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.WorkspaceID != wsID {
		t.Fatalf("expected workspaceID %v, got %v", wsID, s.WorkspaceID)
	}
	if s.Locale != domain.LocaleTR || s.Currency != money.TRY || s.Palette != domain.PaletteNavy {
		t.Fatalf("defaults mismatch: %+v", s)
	}
}

func TestService_UpdateSettingsValidation(t *testing.T) {
	svc := app.NewService(newFakeStore())
	wsID := id.New()

	// Invalid palette
	_, err := svc.UpdateSettings(context.Background(), wsID, app.UpdateSettingsInput{
		Palette: ptr("invalid-color"),
	})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for palette, got %v", err)
	}

	// Invalid locale
	_, err = svc.UpdateSettings(context.Background(), wsID, app.UpdateSettingsInput{
		Locale: ptr("fr"),
	})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for locale, got %v", err)
	}

	// Invalid currency
	_, err = svc.UpdateSettings(context.Background(), wsID, app.UpdateSettingsInput{
		Currency: ptr("BTC"),
	})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for currency, got %v", err)
	}
}

func TestService_UpdateAndGetRoundtrip(t *testing.T) {
	svc := app.NewService(newFakeStore())
	wsID := id.New()

	updated, err := svc.UpdateSettings(context.Background(), wsID, app.UpdateSettingsInput{
		Name:     ptr("Dentist Deluxe"),
		Locale:   ptr("en"),
		Currency: ptr("EUR"),
		Palette:  ptr("forest"),
		Phone:    ptr("+90 555 111 2233"),
		MapURL:   ptr("https://maps.example.com"),
		Intro:    ptr("Welcome to our clinic"),
		Legal:    ptr("Legal notice text"),
		LogoKey:  ptr("logo_123.png"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Name != "Dentist Deluxe" || updated.Locale != domain.LocaleEN || updated.Currency != money.EUR || updated.Palette != domain.PaletteForest {
		t.Fatalf("updated fields mismatch: %+v", updated)
	}

	got, err := svc.GetSettings(context.Background(), wsID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != updated {
		t.Fatalf("GET mismatch after PATCH: got %+v, want %+v", got, updated)
	}
}

func TestService_GetProducts(t *testing.T) {
	svc := app.NewService(newFakeStore())
	prods := svc.GetProducts(context.Background())
	if len(prods) != 2 {
		t.Fatalf("expected 2 products, got %d", len(prods))
	}
	if prods[0].ID != "classic" || !prods[0].Open {
		t.Fatalf("classic must be open")
	}
	if prods[1].ID != "ultimate" || prods[1].Open {
		t.Fatalf("ultimate must be closed")
	}
}
