-- name: ClinicGetSettings :one
SELECT workspace_id, name, locale, currency, palette, logo_key, phone, map_url, intro, legal, created_at, updated_at
FROM clinic_settings
WHERE workspace_id = sqlc.arg('workspace_id');

-- name: ClinicUpsertSettings :one
INSERT INTO clinic_settings (
    workspace_id, name, locale, currency, palette, logo_key, phone, map_url, intro, legal
) VALUES (
    sqlc.arg('workspace_id'),
    sqlc.arg('name'),
    sqlc.arg('locale'),
    sqlc.arg('currency'),
    sqlc.arg('palette'),
    sqlc.arg('logo_key'),
    sqlc.arg('phone'),
    sqlc.arg('map_url'),
    sqlc.arg('intro'),
    sqlc.arg('legal')
)
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
RETURNING workspace_id, name, locale, currency, palette, logo_key, phone, map_url, intro, legal, created_at, updated_at;
