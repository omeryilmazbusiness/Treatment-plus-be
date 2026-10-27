// Package domain is the clinic brand/settings context. No patient PHI.
package domain

import (
	"strings"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Locale string

const (
	LocaleTR Locale = "tr"
	LocaleEN Locale = "en"
)

func ParseLocale(s string) (Locale, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(LocaleTR):
		return LocaleTR, nil
	case string(LocaleEN):
		return LocaleEN, nil
	default:
		return "", errx.New("invalid_locale", "unsupported locale (must be tr or en)", errx.ErrInvalid)
	}
}

type Palette string

const (
	PaletteNavy   Palette = "navy"
	PaletteForest Palette = "forest"
	PaletteRose   Palette = "rose"
	PaletteSlate  Palette = "slate"
)

func ParsePalette(s string) (Palette, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(PaletteNavy):
		return PaletteNavy, nil
	case string(PaletteForest):
		return PaletteForest, nil
	case string(PaletteRose):
		return PaletteRose, nil
	case string(PaletteSlate):
		return PaletteSlate, nil
	default:
		return "", errx.New("invalid_palette", "unsupported palette (must be navy, forest, rose, or slate)", errx.ErrInvalid)
	}
}

func ParseCurrency(s string) (money.Currency, error) {
	c := money.Currency(strings.ToUpper(strings.TrimSpace(s)))
	if !money.ValidCurrency(c) {
		return "", errx.New("invalid_currency", "unsupported currency (must be TRY, EUR, or USD)", errx.ErrInvalid)
	}
	return c, nil
}

type Settings struct {
	WorkspaceID id.ID
	Name        string
	Locale      Locale
	Currency    money.Currency
	Palette     Palette
	LogoKey     string
	Phone       string
	MapURL      string
	Intro       string
	Legal       string
}

func (s Settings) ClinicID() id.ID {
	return s.WorkspaceID
}

func DefaultSettings(workspaceID id.ID, name string) Settings {
	return Settings{
		WorkspaceID: workspaceID,
		Name:        name,
		Locale:      LocaleTR,
		Currency:    money.TRY,
		Palette:     PaletteNavy,
		LogoKey:     "",
		Phone:       "",
		MapURL:      "",
		Intro:       "",
		Legal:       "",
	}
}

type Product struct {
	ID   string `json:"id"`
	Open bool   `json:"open"`
}

func AvailableProducts() []Product {
	return []Product{
		{ID: "classic", Open: true},
		{ID: "ultimate", Open: false},
	}
}
