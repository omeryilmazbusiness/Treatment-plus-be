package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/rls"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) GetView(ctx context.Context, clinicID, planID id.ID) (domain.PlanView, error) {
	var view domain.PlanView
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var status string
		var payload []byte
		row := tx.QueryRow(ctx, `
			SELECT status, snapshot_json
			FROM plans
			WHERE workspace_id = $1 AND id = $2
		`, toUUID(clinicID), toUUID(planID))
		if err := row.Scan(&status, &payload); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}
		if status == "draft" || len(payload) == 0 {
			return errx.New("plan_not_ready", "plan must be ready before PDF generate", errx.ErrInvalid)
		}
		view = domain.PlanView{PlanID: planID, ClinicID: clinicID, Payload: payload}
		return nil
	})
	return view, err
}

func (s *Store) SaveDocument(ctx context.Context, doc domain.Document) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO plan_documents (plan_id, workspace_id, content_type, page_count, byte_size, bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (plan_id) DO UPDATE SET
				content_type = EXCLUDED.content_type,
				page_count = EXCLUDED.page_count,
				byte_size = EXCLUDED.byte_size,
				bytes = EXCLUDED.bytes,
				updated_at = now()
		`, toUUID(doc.PlanID), toUUID(doc.WorkspaceID), doc.ContentType, doc.PageCount, len(doc.Bytes), doc.Bytes)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}
		return nil
	})
}

func (s *Store) GetDocument(ctx context.Context, workspaceID, planID id.ID) (domain.Document, error) {
	var out domain.Document
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var pid, ws pgtype.UUID
		row := tx.QueryRow(ctx, `
			SELECT plan_id, workspace_id, content_type, page_count, bytes, created_at
			FROM plan_documents
			WHERE workspace_id = $1 AND plan_id = $2
		`, toUUID(workspaceID), toUUID(planID))
		if err := row.Scan(&pid, &ws, &out.ContentType, &out.PageCount, &out.Bytes, &out.CreatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}
		out.PlanID = fromUUID(pid)
		out.WorkspaceID = fromUUID(ws)
		return nil
	})
	return out, err
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
