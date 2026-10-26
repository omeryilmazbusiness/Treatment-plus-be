package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type PlanReader interface {
	GetView(ctx context.Context, clinicID, planID id.ID) (domain.PlanView, error)
}

type PDFRenderer interface {
	Render(ctx context.Context, view domain.PlanView) ([]byte, error)
}
