package app

import (
	"context"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/media/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type ObjectStore interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (url string, err error)
	PresignGet(ctx context.Context, key string, ttl time.Duration) (url string, err error)
}

type AssetRepository interface {
	Save(ctx context.Context, asset domain.Asset) error
	Get(ctx context.Context, clinicID, assetID id.ID) (domain.Asset, error)
}
