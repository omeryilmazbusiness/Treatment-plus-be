package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type SettingsStore interface {
	GetSettings(ctx context.Context, workspaceID id.ID) (domain.Settings, error)
	SaveSettings(ctx context.Context, settings domain.Settings) error
}

type SubscriptionReader interface {
	GetActiveTier(ctx context.Context, wsID id.ID) (tier string, allowsUltimate bool, err error)
}
