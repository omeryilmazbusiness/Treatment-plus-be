package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type PlanReader interface {
	Get(ctx context.Context, clinicID, planID id.ID) (domain.Plan, error)
}

type PlanWriter interface {
	Save(ctx context.Context, plan domain.Plan) error
}

type Totals interface {
	Compute(plan domain.Plan) (domain.Totals, error)
}
