package domain

import (
	"strings"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

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
