package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/media/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type AssetStore interface {
	SaveAsset(ctx context.Context, asset domain.Asset) error
	GetAsset(ctx context.Context, workspaceID id.ID, key string) (domain.Asset, error)
}
