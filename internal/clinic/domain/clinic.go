// Package domain is the clinic brand/settings context. No patient PHI.
package domain

import (
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Locale string

const (
	LocaleTR Locale = "tr"
	LocaleEN Locale = "en"
)

type Settings struct {
	ClinicID    id.ID
	Name        string
	Locale      Locale
	Currency    money.Currency
	Phone       string
	MapURL      string
	Intro       string
	Legal       string
	PrimaryColor string
	LogoKey     string
}
