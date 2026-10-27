// Package domain is the identity bounded context.
// The only actor is a clinic account. No staff roles, invites, or patient logins.
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

const (
	NameMinRunes     = 2
	NameMaxRunes     = 120
	PasswordMinRunes = 8
	PasswordMaxRunes = 128
	RefreshSecretLen = 32
)

type Clinic struct {
	ID           id.ID
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID          id.ID
	ClinicID    id.ID
	RefreshHash []byte
	FamilyID    id.ID
	ReusedAt    *time.Time
	RevokedAt   *time.Time
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

func (s Session) Reused() bool  { return s.ReusedAt != nil }
func (s Session) Revoked() bool { return s.RevokedAt != nil }

func (s Session) Expired(now time.Time) bool {
	return !s.ExpiresAt.After(now)
}

// Usable is the happy path: not reused, not revoked, not expired.
func (s Session) Usable(now time.Time) error {
	if s.Revoked() {
		return errx.ErrUnauthenticated
	}
	if s.Expired(now) {
		return errx.ErrUnauthenticated
	}
	if s.Reused() {
		return ErrRefreshReused
	}
	return nil
}

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(collapseSpace(name))
	n := utf8.RuneCountInString(name)
	if n < NameMinRunes || n > NameMaxRunes {
		return "", errx.New("invalid_name", "clinic name must be 2–120 characters", errx.ErrInvalid)
	}
	return name, nil
}

func NormalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", errx.New("invalid_email", "email is required", errx.ErrInvalid)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", errx.New("invalid_email", "email is invalid", errx.ErrInvalid)
	}
	return email, nil
}

func ValidatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < PasswordMinRunes || n > PasswordMaxRunes {
		return errx.New("invalid_password", "password must be 8–128 characters", errx.ErrInvalid)
	}
	return nil
}

func NewRefreshSecret() (raw string, hash []byte, err error) {
	buf := make([]byte, RefreshSecretLen)
	if _, err = rand.Read(buf); err != nil {
		return "", nil, err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

func HashRefresh(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func collapseSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return b.String()
}
