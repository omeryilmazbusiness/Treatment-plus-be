package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/ai/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Writer interface {
	Draft(ctx context.Context, job domain.Job) (text string, err error)
}

type JobQueue interface {
	Enqueue(ctx context.Context, job domain.Job) error
	Get(ctx context.Context, jobID id.ID) (domain.Job, error)
}
