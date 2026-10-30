package pg

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/rls"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/subscription/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListPlans(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tier, name, description, billing_interval, price_cents, currency,
		       plan_quota, has_3d_viz, has_patient_portal, has_telemetry,
		       has_digital_signature, has_onboarding_session, features, is_active, created_at
		FROM subscription_plans
		WHERE is_active = true
		ORDER BY price_cents ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.SubscriptionPlan
	for rows.Next() {
		var (
			p            domain.SubscriptionPlan
			featuresJSON []byte
		)
		if err := rows.Scan(
			&p.ID, &p.Tier, &p.Name, &p.Description, &p.BillingInterval, &p.PriceCents, &p.Currency,
			&p.PlanQuota, &p.Has3DViz, &p.HasPatientPortal, &p.HasTelemetry,
			&p.HasDigitalSignature, &p.HasOnboardingSession, &featuresJSON, &p.IsActive, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(featuresJSON) > 0 {
			_ = json.Unmarshal(featuresJSON, &p.Features)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetPlan(ctx context.Context, planID string) (domain.SubscriptionPlan, error) {
	var (
		p            domain.SubscriptionPlan
		featuresJSON []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, tier, name, description, billing_interval, price_cents, currency,
		       plan_quota, has_3d_viz, has_patient_portal, has_telemetry,
		       has_digital_signature, has_onboarding_session, features, is_active, created_at
		FROM subscription_plans
		WHERE id = $1
	`, planID).Scan(
		&p.ID, &p.Tier, &p.Name, &p.Description, &p.BillingInterval, &p.PriceCents, &p.Currency,
		&p.PlanQuota, &p.Has3DViz, &p.HasPatientPortal, &p.HasTelemetry,
		&p.HasDigitalSignature, &p.HasOnboardingSession, &featuresJSON, &p.IsActive, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SubscriptionPlan{}, errx.New("plan_not_found", "subscription plan not found", errx.ErrNotFound)
		}
		return domain.SubscriptionPlan{}, err
	}
	if len(featuresJSON) > 0 {
		_ = json.Unmarshal(featuresJSON, &p.Features)
	}
	return p, nil
}

func (s *Store) GetClinicSubscription(ctx context.Context, clinicID id.ID) (domain.ClinicSubscription, error) {
	var out domain.ClinicSubscription
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}

		var (
			subIDUUID, wsUUID pgtype.UUID
			canceledAt        *time.Time
			featuresJSON      []byte
		)

		row := tx.QueryRow(ctx, `
			SELECT cs.id, cs.workspace_id, cs.plan_id, cs.status,
			       cs.current_period_start, cs.current_period_end,
			       cs.cancel_at_period_end, cs.canceled_at, cs.plans_created_count,
			       cs.created_at, cs.updated_at,
			       sp.id, sp.tier, sp.name, sp.description, sp.billing_interval,
			       sp.price_cents, sp.currency, sp.plan_quota, sp.has_3d_viz,
			       sp.has_patient_portal, sp.has_telemetry, sp.has_digital_signature,
			       sp.has_onboarding_session, sp.features, sp.is_active, sp.created_at
			FROM clinic_subscriptions cs
			JOIN subscription_plans sp ON sp.id = cs.plan_id
			WHERE cs.workspace_id = $1
		`, toUUID(clinicID))

		err := row.Scan(
			&subIDUUID, &wsUUID, &out.PlanID, &out.Status,
			&out.CurrentPeriodStart, &out.CurrentPeriodEnd,
			&out.CancelAtPeriodEnd, &canceledAt, &out.PlansCreatedCount,
			&out.CreatedAt, &out.UpdatedAt,
			&out.Plan.ID, &out.Plan.Tier, &out.Plan.Name, &out.Plan.Description, &out.Plan.BillingInterval,
			&out.Plan.PriceCents, &out.Plan.Currency, &out.Plan.PlanQuota, &out.Plan.Has3DViz,
			&out.Plan.HasPatientPortal, &out.Plan.HasTelemetry, &out.Plan.HasDigitalSignature,
			&out.Plan.HasOnboardingSession, &featuresJSON, &out.Plan.IsActive, &out.Plan.CreatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.New("subscription_not_found", "clinic has no active subscription", errx.ErrNotFound)
			}
			return err
		}

		out.ID = fromUUID(subIDUUID)
		out.WorkspaceID = fromUUID(wsUUID)
		out.CanceledAt = canceledAt
		if len(featuresJSON) > 0 {
			_ = json.Unmarshal(featuresJSON, &out.Plan.Features)
		}
		return nil
	})
	return out, err
}

func (s *Store) UpsertClinicSubscription(ctx context.Context, sub domain.ClinicSubscription) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO clinic_subscriptions (
				id, workspace_id, plan_id, status, current_period_start, current_period_end,
				cancel_at_period_end, canceled_at, plans_created_count, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
			)
			ON CONFLICT (workspace_id) DO UPDATE SET
				plan_id = EXCLUDED.plan_id,
				status = EXCLUDED.status,
				current_period_start = EXCLUDED.current_period_start,
				current_period_end = EXCLUDED.current_period_end,
				cancel_at_period_end = EXCLUDED.cancel_at_period_end,
				canceled_at = EXCLUDED.canceled_at,
				updated_at = now()
		`,
			toUUID(sub.ID),
			toUUID(sub.WorkspaceID),
			sub.PlanID,
			string(sub.Status),
			sub.CurrentPeriodStart,
			sub.CurrentPeriodEnd,
			sub.CancelAtPeriodEnd,
			sub.CanceledAt,
			sub.PlansCreatedCount,
			sub.CreatedAt,
			sub.UpdatedAt,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}
		return nil
	})
}

func (s *Store) IncrementPlanCount(ctx context.Context, clinicID id.ID) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			UPDATE clinic_subscriptions
			SET plans_created_count = plans_created_count + 1, updated_at = now()
			WHERE workspace_id = $1
		`, toUUID(clinicID))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.New("subscription_not_found", "subscription not found", errx.ErrNotFound)
		}
		return nil
	})
}

func (s *Store) CreateDemoRequest(ctx context.Context, req domain.DemoRequest) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO demo_requests (id, website_url, clinic_name, contact_name, email, phone, notes, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		toUUID(req.ID),
		req.WebsiteURL,
		req.ClinicName,
		req.ContactName,
		req.Email,
		req.Phone,
		req.Notes,
		req.Status,
		req.CreatedAt,
	)
	return err
}

func (s *Store) ListDemoRequests(ctx context.Context) ([]domain.DemoRequest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, website_url, clinic_name, contact_name, email, phone, notes, status, created_at
		FROM demo_requests
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.DemoRequest
	for rows.Next() {
		var (
			r     domain.DemoRequest
			uUUID pgtype.UUID
		)
		if err := rows.Scan(
			&uUUID, &r.WebsiteURL, &r.ClinicName, &r.ContactName, &r.Email, &r.Phone, &r.Notes, &r.Status, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		r.ID = fromUUID(uUUID)
		out = append(out, r)
	}
	return out, rows.Err()
}

func toUUID(v id.ID) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(v.UUIDString())
	return u
}

func fromUUID(u pgtype.UUID) id.ID {
	if !u.Valid {
		return id.ID{}
	}
	out, err := id.FromBytes(u.Bytes[:])
	if err != nil {
		return id.ID{}
	}
	return out
}
