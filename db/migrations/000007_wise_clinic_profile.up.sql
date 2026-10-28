-- Wise-style clinic brand: contact, welcome, about, and JSON page blocks.

ALTER TABLE clinic_settings
    ADD COLUMN slogan text NOT NULL DEFAULT '',
    ADD COLUMN email text NOT NULL DEFAULT '',
    ADD COLUMN website text NOT NULL DEFAULT '',
    ADD COLUMN address text NOT NULL DEFAULT '',
    ADD COLUMN hours text NOT NULL DEFAULT '',
    ADD COLUMN cover_image_key text NOT NULL DEFAULT '',
    ADD COLUMN about_image_key text NOT NULL DEFAULT '',
    ADD COLUMN about_text text NOT NULL DEFAULT '',
    ADD COLUMN welcome_body text NOT NULL DEFAULT '',
    ADD COLUMN signer_name text NOT NULL DEFAULT '',
    ADD COLUMN signer_title text NOT NULL DEFAULT '',
    ADD COLUMN highlights_json jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN doctor_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN lab_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN awards_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN hotel_json jsonb NOT NULL DEFAULT '{}'::jsonb;
