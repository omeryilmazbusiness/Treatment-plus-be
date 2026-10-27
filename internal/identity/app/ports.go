package app

import (
	"context"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type WorkspaceReader interface {
	GetVisible(ctx context.Context, id id.ID) (domain.Workspace, error)
}

type AccountLookup interface {
	FindByEmail(ctx context.Context, email string) (domain.Account, error)
	FindByWorkspaceID(ctx context.Context, workspaceID id.ID) (domain.Account, error)
}

type SessionReader interface {
	GetByRefreshHash(ctx context.Context, hash []byte) (domain.Session, error)
}

type WriteTx interface {
	CreateAccount(ctx context.Context, account domain.Account) error
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
	Issue(userID, workspaceID id.ID, role domain.Role, now time.Time) (token string, ttl time.Duration, err error)
}
