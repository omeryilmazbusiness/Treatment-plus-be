package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type ShareRepository interface {
	Save(ctx context.Context, link domain.Link) error
	GetByTokenHash(ctx context.Context, hash []byte) (domain.Link, error)
	Revoke(ctx context.Context, clinicID, linkID id.ID) error
}

type TokenHasher interface {
	Hash(raw string) []byte
}

type ShareIssuer interface {
	Issue(ctx context.Context, clinicID, planID id.ID) (rawToken string, err error)
}
