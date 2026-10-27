package app

import (
	"context"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

// Split readers/writers: no god repository (ISP).

// ClinicReader is tenant-scoped. Postgres applies RLS (0 rows for another clinic).
type ClinicReader interface {
	GetVisible(ctx context.Context, id id.ID) (domain.Clinic, error)
}

// ClinicLookup is auth bootstrap (SECURITY DEFINER). Used before a tenant is in context.
type ClinicLookup interface {
	FindByEmail(ctx context.Context, email string) (domain.Clinic, error)
	FindByID(ctx context.Context, id id.ID) (domain.Clinic, error)
}

type SessionReader interface {
	GetByRefreshHash(ctx context.Context, hash []byte) (domain.Session, error)
}

type WriteTx interface {
	CreateClinic(ctx context.Context, clinic domain.Clinic) error
	SaveSession(ctx context.Context, session domain.Session) error
	MarkReused(ctx context.Context, sessionID id.ID, at time.Time) error
	RevokeFamily(ctx context.Context, familyID id.ID, at time.Time) error
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, tx WriteTx) error) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) error
}

type AccessIssuer interface {
	Issue(clinicID id.ID, now time.Time) (token string, ttl time.Duration, err error)
}
