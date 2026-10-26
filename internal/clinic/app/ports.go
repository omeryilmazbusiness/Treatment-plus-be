package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type ClinicRepository interface {
	Get(ctx context.Context, id id.ID) (domain.Settings, error)
}

type BrandStore interface {
	GetSettings(ctx context.Context, clinicID id.ID) (domain.Settings, error)
	SaveSettings(ctx context.Context, settings domain.Settings) error
}
