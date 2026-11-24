package app

import (
	"context"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/domain"
)

type SubscriptionStore interface {
	ListPlans(ctx context.Context) ([]domain.SubscriptionPlan, error)
	GetPlan(ctx context.Context, planID string) (domain.SubscriptionPlan, error)
	GetClinicSubscription(ctx context.Context, clinicID id.ID) (domain.ClinicSubscription, error)
	UpsertClinicSubscription(ctx context.Context, sub domain.ClinicSubscription) error
	IncrementPlanCount(ctx context.Context, clinicID id.ID) error
	// ResetUsage starts a new monthly usage window with a zero counter.
	ResetUsage(ctx context.Context, clinicID id.ID, periodStart time.Time) error
	CreateDemoRequest(ctx context.Context, req domain.DemoRequest) error
	ListDemoRequests(ctx context.Context) ([]domain.DemoRequest, error)
}

