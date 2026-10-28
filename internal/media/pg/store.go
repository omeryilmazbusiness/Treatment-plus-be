package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/treatment-plus/treatment-plus-be/internal/media/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/rls"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) SaveAsset(ctx context.Context, asset domain.Asset) error {
	if err := asset.Validate(); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO media_assets (id, workspace_id, object_key, kind, content_type, bytes, byte_size)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (workspace_id, object_key) DO UPDATE SET
				kind = EXCLUDED.kind,
				content_type = EXCLUDED.content_type,
				bytes = EXCLUDED.bytes,
				byte_size = EXCLUDED.byte_size
		`, toUUID(asset.ID), toUUID(asset.WorkspaceID), asset.Key, string(asset.Kind), asset.ContentType, asset.Bytes, len(asset.Bytes))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}
		return nil
	})
}

func (s *Store) GetAsset(ctx context.Context, workspaceID id.ID, key string) (domain.Asset, error) {
	var out domain.Asset
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var aid, ws pgtype.UUID
		row := tx.QueryRow(ctx, `
			SELECT id, workspace_id, object_key, kind, content_type, bytes
			FROM media_assets
			WHERE workspace_id = $1 AND object_key = $2
		`, toUUID(workspaceID), key)
		if err := row.Scan(&aid, &ws, &out.Key, &out.Kind, &out.ContentType, &out.Bytes); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}
		out.ID = fromUUID(aid)
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
