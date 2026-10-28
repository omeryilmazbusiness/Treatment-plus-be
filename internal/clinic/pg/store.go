package pg

import (
	"context"
	"encoding/json"
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
			wsUUID                                   pgtype.UUID
			name, slogan, locale, currency, palette  string
			logoKey, coverKey, phone, email, website string
			address, hours, mapURL, intro, legal     string
			aboutText, aboutKey, welcomeBody         string
			signerName, signerTitle                  string
			highlightsJSON, doctorJSON, labJSON      []byte
			awardsJSON, hotelJSON                    []byte
		)

		row := tx.QueryRow(ctx, `
			SELECT workspace_id, name, slogan, locale, currency, palette,
			       logo_key, cover_image_key, phone, email, website, address, hours,
			       map_url, intro, legal, about_text, about_image_key, welcome_body,
			       signer_name, signer_title,
			       highlights_json, doctor_json, lab_json, awards_json, hotel_json
			FROM clinic_settings
			WHERE workspace_id = $1
		`, toUUID(workspaceID))

		err := row.Scan(
			&wsUUID, &name, &slogan, &locale, &currency, &palette,
			&logoKey, &coverKey, &phone, &email, &website, &address, &hours,
			&mapURL, &intro, &legal, &aboutText, &aboutKey, &welcomeBody,
			&signerName, &signerTitle,
			&highlightsJSON, &doctorJSON, &labJSON, &awardsJSON, &hotelJSON,
		)
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
			Slogan:      slogan,
			Locale:      loc,
			Currency:    curr,
			Palette:     pal,
			LogoKey:     logoKey,
			CoverKey:    coverKey,
			Phone:       phone,
			Email:       email,
			Website:     website,
			Address:     address,
			Hours:       hours,
			MapURL:      mapURL,
			Intro:       intro,
			Legal:       legal,
			AboutText:   aboutText,
			AboutKey:    aboutKey,
			WelcomeBody: welcomeBody,
			SignerName:  signerName,
			SignerTitle: signerTitle,
			Highlights:  decodeJSON[[]domain.Stat](highlightsJSON),
			Doctor:      decodeJSON[domain.Doctor](doctorJSON),
			Lab:         decodeJSON[domain.Lab](labJSON),
			Awards:      decodeJSON[domain.Awards](awardsJSON),
			Hotel:       decodeJSON[domain.Hotel](hotelJSON),
		}
		return nil
	})
	return out, err
}

func (s *Store) SaveSettings(ctx context.Context, settings domain.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	highlightsJSON, err := json.Marshal(settings.Highlights)
	if err != nil {
		return err
	}
	if settings.Highlights == nil {
		highlightsJSON = []byte("[]")
	}
	doctorJSON, err := json.Marshal(settings.Doctor)
	if err != nil {
		return err
	}
	labJSON, err := json.Marshal(settings.Lab)
	if err != nil {
		return err
	}
	awardsJSON, err := json.Marshal(settings.Awards)
	if err != nil {
		return err
	}
	hotelJSON, err := json.Marshal(settings.Hotel)
	if err != nil {
		return err
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO clinic_settings (
				workspace_id, name, slogan, locale, currency, palette,
				logo_key, cover_image_key, phone, email, website, address, hours,
				map_url, intro, legal, about_text, about_image_key, welcome_body,
				signer_name, signer_title,
				highlights_json, doctor_json, lab_json, awards_json, hotel_json
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12, $13,
				$14, $15, $16, $17, $18, $19,
				$20, $21,
				$22, $23, $24, $25, $26
			)
			ON CONFLICT (workspace_id) DO UPDATE SET
				name = EXCLUDED.name,
				slogan = EXCLUDED.slogan,
				locale = EXCLUDED.locale,
				currency = EXCLUDED.currency,
				palette = EXCLUDED.palette,
				logo_key = EXCLUDED.logo_key,
				cover_image_key = EXCLUDED.cover_image_key,
				phone = EXCLUDED.phone,
				email = EXCLUDED.email,
				website = EXCLUDED.website,
				address = EXCLUDED.address,
				hours = EXCLUDED.hours,
				map_url = EXCLUDED.map_url,
				intro = EXCLUDED.intro,
				legal = EXCLUDED.legal,
				about_text = EXCLUDED.about_text,
				about_image_key = EXCLUDED.about_image_key,
				welcome_body = EXCLUDED.welcome_body,
				signer_name = EXCLUDED.signer_name,
				signer_title = EXCLUDED.signer_title,
				highlights_json = EXCLUDED.highlights_json,
				doctor_json = EXCLUDED.doctor_json,
				lab_json = EXCLUDED.lab_json,
				awards_json = EXCLUDED.awards_json,
				hotel_json = EXCLUDED.hotel_json,
				updated_at = now()
		`,
			toUUID(settings.WorkspaceID),
			settings.Name,
			settings.Slogan,
			string(settings.Locale),
			string(settings.Currency),
			string(settings.Palette),
			settings.LogoKey,
			settings.CoverKey,
			settings.Phone,
			settings.Email,
			settings.Website,
			settings.Address,
			settings.Hours,
			settings.MapURL,
			settings.Intro,
			settings.Legal,
			settings.AboutText,
			settings.AboutKey,
			settings.WelcomeBody,
			settings.SignerName,
			settings.SignerTitle,
			highlightsJSON,
			doctorJSON,
			labJSON,
			awardsJSON,
			hotelJSON,
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

func decodeJSON[T any](raw []byte) T {
	var out T
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func toUUID(v id.ID) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(v.UUIDString())
	return u
}
