// Package money is the shared money type. Amount is integer minor units; no float.
package money

import (
	"fmt"
	"strings"
)

// Currency is an ISO 4217 code. v1 set is closed.
type Currency string

const (
	TRY Currency = "TRY"
	EUR Currency = "EUR"
	USD Currency = "USD"
)

// Money is an amount in minor units (kuruş / cent) plus currency.
type Money struct {
	Amount   int64
	Currency Currency
}

func New(amount int64, currency Currency) (Money, error) {
	if !ValidCurrency(currency) {
		return Money{}, fmt.Errorf("money: unsupported currency %q", currency)
	}
	return Money{Amount: amount, Currency: currency}, nil
}

func Zero(currency Currency) Money {
	return Money{Amount: 0, Currency: currency}
}

func ValidCurrency(c Currency) bool {
	switch Currency(strings.ToUpper(string(c))) {
	case TRY, EUR, USD:
		return true
	default:
		return false
	}
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("money: currency mismatch %s vs %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("money: currency mismatch %s vs %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount - other.Amount, Currency: m.Currency}, nil
}

func (m Money) IsZero() bool { return m.Amount == 0 }
