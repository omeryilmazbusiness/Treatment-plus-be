-- Phase 2: Clinic Settings (Brand & PDF Template)
-- 1 workspace = 1 row in clinic_settings. No patient PHI.

CREATE TABLE clinic_settings (
    workspace_id uuid PRIMARY KEY REFERENCES clinics (id) ON DELETE CASCADE,
    name text NOT NULL DEFAULT '',
    locale text NOT NULL DEFAULT 'tr',
    currency text NOT NULL DEFAULT 'TRY',
    palette text NOT NULL DEFAULT 'navy',
    logo_key text NOT NULL DEFAULT '',
    phone text NOT NULL DEFAULT '',
    map_url text NOT NULL DEFAULT '',
    intro text NOT NULL DEFAULT '',
    legal text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT clinic_settings_locale_check CHECK (locale IN ('tr', 'en')),
    CONSTRAINT clinic_settings_currency_check CHECK (currency IN ('TRY', 'EUR', 'USD')),
    CONSTRAINT clinic_settings_palette_check CHECK (palette IN ('navy', 'forest', 'rose', 'slate'))
);

CREATE TRIGGER clinic_settings_set_updated_at
    BEFORE UPDATE ON clinic_settings
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE clinic_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE clinic_settings FORCE ROW LEVEL SECURITY;

-- Tenant isolation: missing or mismatched app.clinic_id yields 0 rows.
CREATE POLICY clinic_settings_tenant ON clinic_settings
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON clinic_settings TO treatment_app;
