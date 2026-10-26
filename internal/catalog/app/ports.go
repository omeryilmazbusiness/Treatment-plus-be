package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/catalog/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type CatalogReader interface {
	ListDiagnoses(ctx context.Context, clinicID id.ID) ([]domain.Diagnosis, error)
	ListTreatments(ctx context.Context, clinicID id.ID) ([]domain.Treatment, error)
}

type CatalogWriter interface {
	SaveTreatment(ctx context.Context, clinicID id.ID, t domain.Treatment) error
}

type PriceListReader interface {
	Get(ctx context.Context, clinicID, listID id.ID) (domain.PriceList, error)
}
