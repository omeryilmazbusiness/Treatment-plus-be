package domain

import (
	"strings"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

// Role is the account kind. Authorization reads this; it is never taken from the body after login.
type Role string

const (
	RoleClinic   Role = "clinic"
	RolePersonal Role = "personal"
)

func ParseRole(raw string) (Role, error) {
	switch Role(strings.ToLower(strings.TrimSpace(raw))) {
	case RoleClinic:
		return RoleClinic, nil
	case RolePersonal:
		return RolePersonal, nil
	default:
		return "", errx.New("invalid_role", "role must be clinic or personal", errx.ErrInvalid)
	}
}

func (r Role) Valid() bool {
	return r == RoleClinic || r == RolePersonal
}
