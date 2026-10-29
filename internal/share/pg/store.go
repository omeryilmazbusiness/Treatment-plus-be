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
	"github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateLink(ctx context.Context, link domain.Link) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO share_links (
				id, workspace_id, plan_id, token_hash, token_prefix, expires_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			toUUID(link.ID),
			toUUID(link.ClinicID),
			toUUID(link.PlanID),
			link.TokenHash,
			link.TokenPrefix,
			link.ExpiresAt,
			link.CreatedAt,
			link.UpdatedAt,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}

		// Also update plan status to 'sent' if it was 'ready'
		_, _ = tx.Exec(ctx, `
			UPDATE plans
			SET status = 'sent', updated_at = now()
			WHERE workspace_id = $1 AND id = $2 AND status = 'ready'
		`, toUUID(link.ClinicID), toUUID(link.PlanID))

		return nil
	})
}

func (s *Store) GetLink(ctx context.Context, clinicID, linkID id.ID) (domain.Link, error) {
	var l domain.Link
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var (
			idUUID, wsUUID, plUUID              pgtype.UUID
			tokenHash                           []byte
			tokenPrefix                         string
			expiresAt, createdAt, updatedAt     time.Time
			revokedAt, lastViewedAt, acceptedAt *time.Time
			viewsCount                          int
			acceptedSig, acceptedIP             string
		)
		row := tx.QueryRow(ctx, `
			SELECT id, workspace_id, plan_id, token_hash, token_prefix,
			       expires_at, revoked_at, views_count, last_viewed_at,
			       accepted_at, accepted_signature, accepted_ip,
			       created_at, updated_at
			FROM share_links
			WHERE workspace_id = $1 AND id = $2
		`, toUUID(clinicID), toUUID(linkID))

		err := row.Scan(
			&idUUID, &wsUUID, &plUUID, &tokenHash, &tokenPrefix,
			&expiresAt, &revokedAt, &viewsCount, &lastViewedAt,
			&acceptedAt, &acceptedSig, &acceptedIP,
			&createdAt, &updatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}

		l = domain.Link{
			ID:                fromUUID(idUUID),
			ClinicID:          fromUUID(wsUUID),
			PlanID:            fromUUID(plUUID),
			TokenHash:         tokenHash,
			TokenPrefix:       tokenPrefix,
			ExpiresAt:         expiresAt,
			RevokedAt:         revokedAt,
			ViewsCount:        viewsCount,
			LastViewedAt:      lastViewedAt,
			AcceptedAt:        acceptedAt,
			AcceptedSignature: acceptedSig,
			AcceptedIP:        acceptedIP,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}
		return nil
	})
	return l, err
}

func (s *Store) ListLinks(ctx context.Context, clinicID, planID id.ID) ([]domain.Link, error) {
	var out []domain.Link
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT id, workspace_id, plan_id, token_hash, token_prefix,
			       expires_at, revoked_at, views_count, last_viewed_at,
			       accepted_at, accepted_signature, accepted_ip,
			       created_at, updated_at
			FROM share_links
			WHERE workspace_id = $1 AND plan_id = $2
			ORDER BY created_at DESC
		`, toUUID(clinicID), toUUID(planID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				idUUID, wsUUID, plUUID              pgtype.UUID
				tokenHash                           []byte
				tokenPrefix                         string
				expiresAt, createdAt, updatedAt     time.Time
				revokedAt, lastViewedAt, acceptedAt *time.Time
				viewsCount                          int
				acceptedSig, acceptedIP             string
			)
			if err := rows.Scan(
				&idUUID, &wsUUID, &plUUID, &tokenHash, &tokenPrefix,
				&expiresAt, &revokedAt, &viewsCount, &lastViewedAt,
				&acceptedAt, &acceptedSig, &acceptedIP,
				&createdAt, &updatedAt,
			); err != nil {
				return err
			}
			out = append(out, domain.Link{
				ID:                fromUUID(idUUID),
				ClinicID:          fromUUID(wsUUID),
				PlanID:            fromUUID(plUUID),
				TokenHash:         tokenHash,
				TokenPrefix:       tokenPrefix,
				ExpiresAt:         expiresAt,
				RevokedAt:         revokedAt,
				ViewsCount:        viewsCount,
				LastViewedAt:      lastViewedAt,
				AcceptedAt:        acceptedAt,
				AcceptedSignature: acceptedSig,
				AcceptedIP:        acceptedIP,
				CreatedAt:         createdAt,
				UpdatedAt:         updatedAt,
			})
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) RevokeLink(ctx context.Context, clinicID, linkID id.ID) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE share_links
			SET revoked_at = now(), updated_at = now()
			WHERE workspace_id = $1 AND id = $2 AND revoked_at IS NULL
		`, toUUID(clinicID), toUUID(linkID))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrNotFound
		}
		return nil
	})
}

// ResolveToken calls the SECURITY DEFINER function to securely look up an unauthenticated link.
func (s *Store) ResolveToken(ctx context.Context, tokenHash []byte) (domain.Link, error) {
	var (
		idUUID, wsUUID, plUUID pgtype.UUID
		expiresAt              time.Time
		revokedAt              *time.Time
		viewsCount             int
		lastViewedAt           *time.Time
		acceptedAt             *time.Time
		acceptedSig            string
	)

	row := s.pool.QueryRow(ctx, `
		SELECT link_id, workspace_id, plan_id, expires_at, revoked_at,
		       views_count, last_viewed_at, accepted_at, accepted_signature
		FROM share_resolve_token($1)
	`, tokenHash)

	err := row.Scan(
		&idUUID, &wsUUID, &plUUID, &expiresAt, &revokedAt,
		&viewsCount, &lastViewedAt, &acceptedAt, &acceptedSig,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Link{}, errx.New("invalid_token", "share link not found", errx.ErrNotFound)
		}
		return domain.Link{}, err
	}

	link := domain.Link{
		ID:                fromUUID(idUUID),
		ClinicID:          fromUUID(wsUUID),
		PlanID:            fromUUID(plUUID),
		TokenHash:         tokenHash,
		ExpiresAt:         expiresAt,
		RevokedAt:         revokedAt,
		ViewsCount:        viewsCount,
		LastViewedAt:      lastViewedAt,
		AcceptedAt:        acceptedAt,
		AcceptedSignature: acceptedSig,
	}

	now := time.Now().UTC()
	if link.IsRevoked() {
		return link, errx.New("link_revoked", "this share link has been revoked", errx.ErrForbidden)
	}
	if link.IsExpired(now) {
		return link, errx.New("link_expired", "this share link has expired", errx.ErrForbidden)
	}

	return link, nil
}

func (s *Store) RecordEvent(ctx context.Context, tokenHash []byte, eventID id.ID, kind domain.EventKind, metadata map[string]any) error {
	metaJSON, _ := json.Marshal(metadata)
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT share_record_event($1, $2, $3, $4)
	`, tokenHash, toUUID(eventID), string(kind), metaJSON).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return errx.New("invalid_token", "share link not found or inactive", errx.ErrNotFound)
	}
	return nil
}

func (s *Store) AcceptPlan(ctx context.Context, tokenHash []byte, signature, ip string) error {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT share_accept_plan($1, $2, $3)
	`, tokenHash, signature, ip).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return errx.New("invalid_token", "share link not found or inactive", errx.ErrNotFound)
	}
	return nil
}

func (s *Store) ListEvents(ctx context.Context, clinicID, planID id.ID) ([]domain.Event, error) {
	var out []domain.Event
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT se.id, se.workspace_id, se.link_id, se.kind, se.metadata, se.created_at
			FROM share_events se
			JOIN share_links sl ON sl.id = se.link_id
			WHERE se.workspace_id = $1 AND sl.plan_id = $2
			ORDER BY se.created_at DESC
			LIMIT 100
		`, toUUID(clinicID), toUUID(planID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				evUUID, wsUUID, lkUUID pgtype.UUID
				kindStr                string
				metaJSON               []byte
				createdAt              time.Time
			)
			if err := rows.Scan(&evUUID, &wsUUID, &lkUUID, &kindStr, &metaJSON, &createdAt); err != nil {
				return err
			}
			var meta map[string]any
			if len(metaJSON) > 0 {
				_ = json.Unmarshal(metaJSON, &meta)
			}
			out = append(out, domain.Event{
				ID:        fromUUID(evUUID),
				ClinicID:  fromUUID(wsUUID),
				LinkID:    fromUUID(lkUUID),
				Kind:      domain.ParseEventKind(kindStr),
				Metadata:  meta,
				CreatedAt: createdAt,
			})
		}
		return rows.Err()
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
