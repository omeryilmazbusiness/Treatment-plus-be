package domain_test

import (
	"errors"
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

func TestParseLocale(t *testing.T) {
	cases := []struct {
		input   string
		want    domain.Locale
		wantErr bool
	}{
		{"tr", domain.LocaleTR, false},
		{"TR", domain.LocaleTR, false},
		{"en", domain.LocaleEN, false},
		{"EN", domain.LocaleEN, false},
		{"de", "", true},
		{"", "", true},
	}

	for _, tc := range cases {
		got, err := domain.ParseLocale(tc.input)
		if tc.wantErr {
			if err == nil || !errors.Is(err, errx.ErrInvalid) {
				t.Fatalf("ParseLocale(%q) expected ErrInvalid, got %v", tc.input, err)
			}
		} else {
			if err != nil {
				t.Fatalf("ParseLocale(%q) unexpected error %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ParseLocale(%q) = %v, want %v", tc.input, got, tc.want)
			}
		}
	}
}

func TestParsePalette(t *testing.T) {
	cases := []struct {
		input   string
		want    domain.Palette
		wantErr bool
	}{
		{"navy", domain.PaletteNavy, false},
		{"forest", domain.PaletteForest, false},
		{"rose", domain.PaletteRose, false},
		{"slate", domain.PaletteSlate, false},
		{"NAVY", domain.PaletteNavy, false},
		{"#1e293b", "", true}, // arbitrary hex is forbidden
		{"blue", "", true},
		{"", "", true},
	}

	for _, tc := range cases {
		got, err := domain.ParsePalette(tc.input)
		if tc.wantErr {
			if err == nil || !errors.Is(err, errx.ErrInvalid) {
				t.Fatalf("ParsePalette(%q) expected ErrInvalid, got %v", tc.input, err)
			}
		} else {
			if err != nil {
				t.Fatalf("ParsePalette(%q) unexpected error %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ParsePalette(%q) = %v, want %v", tc.input, got, tc.want)
			}
		}
	}
}

func TestParseCurrency(t *testing.T) {
	cases := []struct {
		input   string
		want    money.Currency
		wantErr bool
	}{
		{"try", money.TRY, false},
		{"TRY", money.TRY, false},
		{"eur", money.EUR, false},
		{"usd", money.USD, false},
		{"GBP", "", true},
		{"", "", true},
	}

	for _, tc := range cases {
		got, err := domain.ParseCurrency(tc.input)
		if tc.wantErr {
			if err == nil || !errors.Is(err, errx.ErrInvalid) {
				t.Fatalf("ParseCurrency(%q) expected ErrInvalid, got %v", tc.input, err)
			}
		} else {
			if err != nil {
				t.Fatalf("ParseCurrency(%q) unexpected error %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ParseCurrency(%q) = %v, want %v", tc.input, got, tc.want)
			}
		}
	}
}

func TestAvailableProducts(t *testing.T) {
	prods := domain.AvailableProducts()
	if len(prods) != 2 {
		t.Fatalf("expected 2 products, got %d", len(prods))
	}
	if prods[0].ID != "classic" || !prods[0].Open {
		t.Fatalf("classic must be open: %+v", prods[0])
	}
	if prods[1].ID != "ultimate" || prods[1].Open {
		t.Fatalf("ultimate must be closed: %+v", prods[1])
	}
}

func TestOnboardRequiresClinicBrand(t *testing.T) {
	wsID := id.New()
	_, err := domain.Onboard(wsID, domain.OnboardInput{Name: "Klinik", RequireBrand: true})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected brand required, got %v", err)
	}
	got, err := domain.Onboard(wsID, domain.OnboardInput{
		Name: "Klinik", Slogan: "Every Smile Matters", Phone: "+90 555 111 2233",
		Email: "info@klinik.test", Address: "Bağdat Cad. No:1", Hours: "09:00-18:00",
		SignerName: "Dr. Ada", Locale: "en", Currency: "EUR", Palette: "rose",
		LogoKey: "logo/a", RequireBrand: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Phone != "+90 555 111 2233" || got.Locale != domain.LocaleEN || got.LogoKey != "logo/a" || got.SignerName != "Dr. Ada" {
		t.Fatalf("%+v", got)
	}
}

func TestSettingsValidate(t *testing.T) {
	wsID := id.New()
	if err := domain.DefaultSettings(wsID, "Klinik").Validate(); err != nil {
		t.Fatalf("defaults must be valid: %v", err)
	}
	bad := domain.Settings{WorkspaceID: wsID, Locale: "fr", Currency: money.TRY, Palette: domain.PaletteNavy}
	if err := bad.Validate(); err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected invalid locale, got %v", err)
	}
	empty := domain.Settings{}
	if err := empty.Validate(); err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected invalid workspace, got %v", err)
	}
}

func TestDefaultSettings(t *testing.T) {
	wsID := id.New()
	s := domain.DefaultSettings(wsID, "Test Clinic")
	if s.WorkspaceID != wsID {
		t.Fatalf("workspaceID mismatch")
	}
	if s.Name != "Test Clinic" {
		t.Fatalf("name mismatch")
	}
	if s.Locale != domain.LocaleTR {
		t.Fatalf("default locale must be tr")
	}
	if s.Currency != money.TRY {
		t.Fatalf("default currency must be TRY")
	}
	if s.Palette != domain.PaletteNavy {
		t.Fatalf("default palette must be navy")
	}
}
