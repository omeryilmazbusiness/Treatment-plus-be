package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/catalog/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type CatalogStore interface {
	ListDiagnoses(ctx context.Context, wsID id.ID, includeInactive bool) ([]domain.Diagnosis, error)
	GetDiagnosis(ctx context.Context, wsID id.ID, diagID id.ID) (domain.Diagnosis, error)
	SaveDiagnosis(ctx context.Context, d domain.Diagnosis) error

	ListTreatments(ctx context.Context, wsID id.ID, includeInactive bool) ([]domain.Treatment, error)
	GetTreatment(ctx context.Context, wsID id.ID, treatID id.ID) (domain.Treatment, error)
	SaveTreatment(ctx context.Context, t domain.Treatment) error

	ListPriceLists(ctx context.Context, wsID id.ID) ([]domain.PriceList, error)
	GetPriceList(ctx context.Context, wsID id.ID, listID id.ID) (domain.PriceList, []domain.PriceItem, error)
	GetDefaultPriceList(ctx context.Context, wsID id.ID) (domain.PriceList, []domain.PriceItem, error)
	SavePriceList(ctx context.Context, pl domain.PriceList) error
	SavePriceItems(ctx context.Context, wsID id.ID, listID id.ID, items []domain.PriceItem) error
}
