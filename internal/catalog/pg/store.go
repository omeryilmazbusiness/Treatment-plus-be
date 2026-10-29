package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/treatment-plus/treatment-plus-be/internal/catalog/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/rls"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListDiagnoses(ctx context.Context, wsID id.ID, includeInactive bool) ([]domain.Diagnosis, error) {
	var out []domain.Diagnosis
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		query := `
			SELECT id, workspace_id, code, name, is_active, created_at, updated_at
			FROM diagnoses
			WHERE workspace_id = $1 `
		if !includeInactive {
			query += ` AND is_active = true `
		}
		query += ` ORDER BY code ASC`

		rows, err := tx.Query(ctx, query, toUUID(wsID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var d domain.Diagnosis
			var dID, wsUUID pgtype.UUID
			if err := rows.Scan(&dID, &wsUUID, &d.Code, &d.Name, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
				return err
			}
			d.ID = fromUUID(dID)
			d.WorkspaceID = fromUUID(wsUUID)
			out = append(out, d)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) GetDiagnosis(ctx context.Context, wsID id.ID, diagID id.ID) (domain.Diagnosis, error) {
	var out domain.Diagnosis
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var dID, wsUUID pgtype.UUID
		row := tx.QueryRow(ctx, `
			SELECT id, workspace_id, code, name, is_active, created_at, updated_at
			FROM diagnoses
			WHERE workspace_id = $1 AND id = $2
		`, toUUID(wsID), toUUID(diagID))
		if err := row.Scan(&dID, &wsUUID, &out.Code, &out.Name, &out.IsActive, &out.CreatedAt, &out.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}
		out.ID = fromUUID(dID)
		out.WorkspaceID = fromUUID(wsUUID)
		return nil
	})
	return out, err
}

func (s *Store) SaveDiagnosis(ctx context.Context, d domain.Diagnosis) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO diagnoses (id, workspace_id, code, name, is_active)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (workspace_id, code) DO UPDATE SET
				name = EXCLUDED.name,
				is_active = EXCLUDED.is_active,
				updated_at = now()
		`, toUUID(d.ID), toUUID(d.WorkspaceID), d.Code, d.Name, d.IsActive)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}
		return nil
	})
}

func (s *Store) ListTreatments(ctx context.Context, wsID id.ID, includeInactive bool) ([]domain.Treatment, error) {
	var out []domain.Treatment
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		query := `
			SELECT id, workspace_id, code, name, is_active, created_at, updated_at
			FROM treatments
			WHERE workspace_id = $1 `
		if !includeInactive {
			query += ` AND is_active = true `
		}
		query += ` ORDER BY code ASC`

		rows, err := tx.Query(ctx, query, toUUID(wsID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var t domain.Treatment
			var tID, wsUUID pgtype.UUID
			if err := rows.Scan(&tID, &wsUUID, &t.Code, &t.Name, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
				return err
			}
			t.ID = fromUUID(tID)
			t.WorkspaceID = fromUUID(wsUUID)
			out = append(out, t)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) GetTreatment(ctx context.Context, wsID id.ID, treatID id.ID) (domain.Treatment, error) {
	var out domain.Treatment
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var tID, wsUUID pgtype.UUID
		row := tx.QueryRow(ctx, `
			SELECT id, workspace_id, code, name, is_active, created_at, updated_at
			FROM treatments
			WHERE workspace_id = $1 AND id = $2
		`, toUUID(wsID), toUUID(treatID))
		if err := row.Scan(&tID, &wsUUID, &out.Code, &out.Name, &out.IsActive, &out.CreatedAt, &out.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}
		out.ID = fromUUID(tID)
		out.WorkspaceID = fromUUID(wsUUID)
		return nil
	})
	return out, err
}

func (s *Store) SaveTreatment(ctx context.Context, t domain.Treatment) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO treatments (id, workspace_id, code, name, is_active)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (workspace_id, code) DO UPDATE SET
				name = EXCLUDED.name,
				is_active = EXCLUDED.is_active,
				updated_at = now()
		`, toUUID(t.ID), toUUID(t.WorkspaceID), t.Code, t.Name, t.IsActive)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}
		return nil
	})
}

func (s *Store) DeleteTreatment(ctx context.Context, wsID id.ID, treatID id.ID) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		// Delete price items for this treatment
		_, err := tx.Exec(ctx, `
			DELETE FROM price_items
			WHERE workspace_id = $1 AND treatment_id = $2
		`, toUUID(wsID), toUUID(treatID))
		if err != nil {
			return err
		}
		// Unlink from plan items so existing plan history remains intact
		_, err = tx.Exec(ctx, `
			UPDATE plan_items
			SET treatment_id = NULL
			WHERE workspace_id = $1 AND treatment_id = $2
		`, toUUID(wsID), toUUID(treatID))
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			DELETE FROM treatments
			WHERE workspace_id = $1 AND id = $2
		`, toUUID(wsID), toUUID(treatID))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrNotFound
		}
		return nil
	})
}

func (s *Store) ListPriceLists(ctx context.Context, wsID id.ID) ([]domain.PriceList, error) {
	var out []domain.PriceList
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT id, workspace_id, name, currency, is_default, is_active, created_at, updated_at
			FROM price_lists
			WHERE workspace_id = $1
			ORDER BY is_default DESC, name ASC
		`, toUUID(wsID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var pl domain.PriceList
			var plID, wsUUID pgtype.UUID
			var curr string
			if err := rows.Scan(&plID, &wsUUID, &pl.Name, &curr, &pl.IsDefault, &pl.IsActive, &pl.CreatedAt, &pl.UpdatedAt); err != nil {
				return err
			}
			pl.ID = fromUUID(plID)
			pl.WorkspaceID = fromUUID(wsUUID)
			pl.Currency = money.Currency(curr)
			out = append(out, pl)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) GetPriceList(ctx context.Context, wsID id.ID, listID id.ID) (domain.PriceList, []domain.PriceItem, error) {
	var pl domain.PriceList
	var items []domain.PriceItem
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var plID, wsUUID pgtype.UUID
		var curr string
		row := tx.QueryRow(ctx, `
			SELECT id, workspace_id, name, currency, is_default, is_active, created_at, updated_at
			FROM price_lists
			WHERE workspace_id = $1 AND id = $2
		`, toUUID(wsID), toUUID(listID))
		if err := row.Scan(&plID, &wsUUID, &pl.Name, &curr, &pl.IsDefault, &pl.IsActive, &pl.CreatedAt, &pl.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}
		pl.ID = fromUUID(plID)
		pl.WorkspaceID = fromUUID(wsUUID)
		pl.Currency = money.Currency(curr)

		itemRows, err := tx.Query(ctx, `
			SELECT id, workspace_id, price_list_id, treatment_id, amount, created_at, updated_at
			FROM price_items
			WHERE workspace_id = $1 AND price_list_id = $2
		`, toUUID(wsID), toUUID(listID))
		if err != nil {
			return err
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var it domain.PriceItem
			var itID, itWsUUID, itPLUUID, itTrUUID pgtype.UUID
			if err := itemRows.Scan(&itID, &itWsUUID, &itPLUUID, &itTrUUID, &it.Amount, &it.CreatedAt, &it.UpdatedAt); err != nil {
				return err
			}
			it.ID = fromUUID(itID)
			it.WorkspaceID = fromUUID(itWsUUID)
			it.PriceListID = fromUUID(itPLUUID)
			it.TreatmentID = fromUUID(itTrUUID)
			items = append(items, it)
		}
		return itemRows.Err()
	})
	return pl, items, err
}

func (s *Store) GetDefaultPriceList(ctx context.Context, wsID id.ID) (domain.PriceList, []domain.PriceItem, error) {
	var pl domain.PriceList
	var items []domain.PriceItem
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var plID, wsUUID pgtype.UUID
		var curr string
		row := tx.QueryRow(ctx, `
			SELECT id, workspace_id, name, currency, is_default, is_active, created_at, updated_at
			FROM price_lists
			WHERE workspace_id = $1 AND is_default = true
			LIMIT 1
		`, toUUID(wsID))
		if err := row.Scan(&plID, &wsUUID, &pl.Name, &curr, &pl.IsDefault, &pl.IsActive, &pl.CreatedAt, &pl.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}
		pl.ID = fromUUID(plID)
		pl.WorkspaceID = fromUUID(wsUUID)
		pl.Currency = money.Currency(curr)

		itemRows, err := tx.Query(ctx, `
			SELECT id, workspace_id, price_list_id, treatment_id, amount, created_at, updated_at
			FROM price_items
			WHERE workspace_id = $1 AND price_list_id = $2
		`, toUUID(wsID), toUUID(pl.ID))
		if err != nil {
			return err
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var it domain.PriceItem
			var itID, itWsUUID, itPLUUID, itTrUUID pgtype.UUID
			if err := itemRows.Scan(&itID, &itWsUUID, &itPLUUID, &itTrUUID, &it.Amount, &it.CreatedAt, &it.UpdatedAt); err != nil {
				return err
			}
			it.ID = fromUUID(itID)
			it.WorkspaceID = fromUUID(itWsUUID)
			it.PriceListID = fromUUID(itPLUUID)
			it.TreatmentID = fromUUID(itTrUUID)
			items = append(items, it)
		}
		return itemRows.Err()
	})
	return pl, items, err
}

func (s *Store) SavePriceList(ctx context.Context, pl domain.PriceList) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		if pl.IsDefault {
			_, _ = tx.Exec(ctx, `
				UPDATE price_lists SET is_default = false WHERE workspace_id = $1
			`, toUUID(pl.WorkspaceID))
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO price_lists (id, workspace_id, name, currency, is_default, is_active)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				currency = EXCLUDED.currency,
				is_default = EXCLUDED.is_default,
				is_active = EXCLUDED.is_active,
				updated_at = now()
		`, toUUID(pl.ID), toUUID(pl.WorkspaceID), pl.Name, string(pl.Currency), pl.IsDefault, pl.IsActive)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}
		return nil
	})
}

func (s *Store) SavePriceItems(ctx context.Context, wsID id.ID, listID id.ID, items []domain.PriceItem) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		for _, it := range items {
			_, err := tx.Exec(ctx, `
				INSERT INTO price_items (id, workspace_id, price_list_id, treatment_id, amount)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (price_list_id, treatment_id) DO UPDATE SET
					amount = EXCLUDED.amount,
					updated_at = now()
			`, toUUID(it.ID), toUUID(wsID), toUUID(listID), toUUID(it.TreatmentID), it.Amount)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func toUUID(v id.ID) pgtype.UUID {
	var u pgtype.UUID
	copy(u.Bytes[:], v.Bytes())
	u.Valid = true
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
