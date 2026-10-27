// Package domain is the identity bounded context.
// A user logs in. Their role (clinic | personal) is the authorization key.
// Each user owns one workspace; RLS still isolates by workspace id (app.clinic_id).
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

type User struct {
	ID           id.ID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// Workspace is the tenant. Clinic and personal accounts both get one.
type Workspace struct {
	ID          id.ID
	OwnerUserID id.ID
	Name        string
	Kind        Role
	CreatedAt   time.Time
}

type Membership struct {
	UserID      id.ID
	WorkspaceID id.ID
	Role        Role
}

type Account struct {
	User      User
	Workspace Workspace
	Role      Role
}

type Session struct {
	ID          id.ID
	UserID      id.ID
	ClinicID    id.ID // workspace tenant; RLS key
	Role        Role
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
		return "", errx.New("invalid_name", "name must be 2–120 characters", errx.ErrInvalid)
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
