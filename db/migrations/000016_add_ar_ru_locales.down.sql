ALTER TABLE clinic_settings
    DROP CONSTRAINT IF EXISTS clinic_settings_locale_check;

ALTER TABLE clinic_settings
    ADD CONSTRAINT clinic_settings_locale_check CHECK (locale IN ('tr', 'en'));

ALTER TABLE plans
    DROP CONSTRAINT IF EXISTS plans_locale_check;

ALTER TABLE plans
    ADD CONSTRAINT plans_locale_check CHECK (locale IN ('tr', 'en'));
