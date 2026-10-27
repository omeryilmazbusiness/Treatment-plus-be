package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/rls"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) GetSettings(ctx context.Context, workspaceID id.ID) (domain.Settings, error) {
	var out domain.Settings
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}

		var (
			wsUUID   pgtype.UUID
			name     string
			locale   string
			currency string
			palette  string
			logoKey  string
			phone    string
			mapURL   string
			intro    string
			legal    string
		)

		row := tx.QueryRow(ctx, `
			SELECT workspace_id, name, locale, currency, palette, logo_key, phone, map_url, intro, legal
			FROM clinic_settings
			WHERE workspace_id = $1
		`, toUUID(workspaceID))

		err := row.Scan(&wsUUID, &name, &locale, &currency, &palette, &logoKey, &phone, &mapURL, &intro, &legal)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}

		loc, err := domain.ParseLocale(locale)
		if err != nil {
			return err
		}
		curr, err := domain.ParseCurrency(currency)
		if err != nil {
			return err
		}
		pal, err := domain.ParsePalette(palette)
		if err != nil {
			return err
		}

		out = domain.Settings{
			WorkspaceID: workspaceID,
			Name:        name,
			Locale:      loc,
			Currency:    curr,
			Palette:     pal,
			LogoKey:     logoKey,
			Phone:       phone,
			MapURL:      mapURL,
			Intro:       intro,
			Legal:       legal,
		}
		return nil
	})
	return out, err
}

func (s *Store) SaveSettings(ctx context.Context, settings domain.Settings) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO clinic_settings (
				workspace_id, name, locale, currency, palette, logo_key, phone, map_url, intro, legal
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (workspace_id) DO UPDATE SET
				name = EXCLUDED.name,
				locale = EXCLUDED.locale,
				currency = EXCLUDED.currency,
				palette = EXCLUDED.palette,
				logo_key = EXCLUDED.logo_key,
				phone = EXCLUDED.phone,
				map_url = EXCLUDED.map_url,
				intro = EXCLUDED.intro,
				legal = EXCLUDED.legal,
				updated_at = now()
		`,
			toUUID(settings.WorkspaceID),
			settings.Name,
			string(settings.Locale),
			string(settings.Currency),
			string(settings.Palette),
			settings.LogoKey,
			settings.Phone,
			settings.MapURL,
			settings.Intro,
			settings.Legal,
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

func toUUID(v id.ID) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(v.UUIDString())
	return u
}
