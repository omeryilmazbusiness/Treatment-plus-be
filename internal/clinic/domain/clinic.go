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
	Slogan      string
	Locale      Locale
	Currency    money.Currency
	Palette     Palette
	LogoKey     string
	CoverKey    string
	Phone       string
	Email       string
	Website     string
	Address     string
	Hours       string
	MapURL      string
	Intro       string
	Legal       string
	AboutText   string
	AboutKey    string
	WelcomeBody string
	SignerName  string
	SignerTitle string
	Highlights  []Stat
	Doctor      Doctor
	Lab         Lab
	Awards      Awards
	Hotel       Hotel
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
	}
}

func (s Settings) Validate() error {
	if s.WorkspaceID.IsZero() {
		return errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	if _, err := ParseLocale(string(s.Locale)); err != nil {
		return err
	}
	if _, err := ParseCurrency(string(s.Currency)); err != nil {
		return err
	}
	if _, err := ParsePalette(string(s.Palette)); err != nil {
		return err
	}
	return nil
}

// OnboardInput is the one-time Wise brand profile collected when the workspace is created.
type OnboardInput struct {
	Name         string
	Slogan       string
	Phone        string
	Email        string
	Website      string
	Address      string
	Hours        string
	Locale       string
	Currency     string
	Palette      string
	LogoKey      string
	CoverKey     string
	MapURL       string
	Intro        string
	Legal        string
	AboutText    string
	AboutKey     string
	WelcomeBody  string
	SignerName   string
	SignerTitle  string
	Highlights   []Stat
	Doctor       Doctor
	Lab          Lab
	Awards       Awards
	Hotel        Hotel
	RequireBrand bool
}

func Onboard(workspaceID id.ID, in OnboardInput) (Settings, error) {
	s := DefaultSettings(workspaceID, strings.TrimSpace(in.Name))
	if in.Locale != "" {
		loc, err := ParseLocale(in.Locale)
		if err != nil {
			return Settings{}, err
		}
		s.Locale = loc
	}
	if in.Currency != "" {
		curr, err := ParseCurrency(in.Currency)
		if err != nil {
			return Settings{}, err
		}
		s.Currency = curr
	}
	if in.Palette != "" {
		pal, err := ParsePalette(in.Palette)
		if err != nil {
			return Settings{}, err
		}
		s.Palette = pal
	}
	s.Slogan = strings.TrimSpace(in.Slogan)
	s.Phone = strings.TrimSpace(in.Phone)
	s.Email = strings.TrimSpace(in.Email)
	s.Website = strings.TrimSpace(in.Website)
	s.Address = strings.TrimSpace(in.Address)
	s.Hours = strings.TrimSpace(in.Hours)
	s.LogoKey = strings.TrimSpace(in.LogoKey)
	s.CoverKey = strings.TrimSpace(in.CoverKey)
	s.MapURL = strings.TrimSpace(in.MapURL)
	s.Intro = strings.TrimSpace(in.Intro)
	s.Legal = strings.TrimSpace(in.Legal)
	s.AboutText = strings.TrimSpace(in.AboutText)
	s.AboutKey = strings.TrimSpace(in.AboutKey)
	s.WelcomeBody = strings.TrimSpace(in.WelcomeBody)
	s.SignerName = strings.TrimSpace(in.SignerName)
	s.SignerTitle = strings.TrimSpace(in.SignerTitle)
	for _, h := range in.Highlights {
		if n := h.Normalize(); n.Value != "" || n.Label != "" {
			s.Highlights = append(s.Highlights, n)
		}
	}
	s.Doctor = in.Doctor.Normalize()
	s.Lab = in.Lab.Normalize()
	s.Awards = in.Awards.Normalize()
	s.Hotel = in.Hotel.Normalize()

	if in.RequireBrand {
		if s.Name == "" {
			return Settings{}, errx.New("clinic_name_required", "clinic name is required", errx.ErrInvalid)
		}
		if s.Slogan == "" {
			return Settings{}, errx.New("clinic_slogan_required", "clinic slogan is required for the PDF cover", errx.ErrInvalid)
		}
		if s.Phone == "" {
			return Settings{}, errx.New("clinic_phone_required", "clinic phone is required for the PDF contact page", errx.ErrInvalid)
		}
		if s.Email == "" {
			return Settings{}, errx.New("clinic_email_required", "clinic email is required for the PDF contact page", errx.ErrInvalid)
		}
		if s.Address == "" {
			return Settings{}, errx.New("clinic_address_required", "clinic address is required for the PDF contact page", errx.ErrInvalid)
		}
		if s.Hours == "" {
			return Settings{}, errx.New("clinic_hours_required", "clinic hours are required for the PDF contact page", errx.ErrInvalid)
		}
		if s.SignerName == "" {
			return Settings{}, errx.New("clinic_signer_required", "welcome letter signer is required", errx.ErrInvalid)
		}
	}
	if err := s.Validate(); err != nil {
		return Settings{}, err
	}
	return s, nil
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
