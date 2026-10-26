package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

// Split readers/writers: no god repository (ISP).

type UserReader interface {
	GetByID(ctx context.Context, id id.ID) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
}

type UserWriter interface {
	Create(ctx context.Context, user domain.User) error
}

type ClinicWriter interface {
	Create(ctx context.Context, clinic domain.Clinic) error
}

type MembershipWriter interface {
	Create(ctx context.Context, m domain.Membership) error
}

type SessionStore interface {
	Save(ctx context.Context, session domain.Session) error
	GetByRefreshHash(ctx context.Context, hash []byte) (domain.Session, error)
	RevokeFamily(ctx context.Context, familyID id.ID) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) error
}
