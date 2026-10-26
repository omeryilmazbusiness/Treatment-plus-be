// Package domain is the identity bounded context.
// Users, memberships, sessions, roles. No plan or price rules.
package domain

import "github.com/treatment-plus/treatment-plus-be/internal/shared/id"

type Role string

const (
	RoleOwner     Role = "owner"
	RoleDoctor    Role = "doctor"
	RoleAssistant Role = "assistant"
)

type User struct {
	ID           id.ID
	Email        string
	PasswordHash string
}

type Clinic struct {
	ID   id.ID
	Name string
}

type Membership struct {
	UserID   id.ID
	ClinicID id.ID
	Role     Role
}

type Session struct {
	ID           id.ID
	UserID       id.ID
	ClinicID     id.ID
	RefreshHash  []byte
	FamilyID     id.ID
}
