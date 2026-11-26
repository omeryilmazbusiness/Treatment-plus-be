ALTER TABLE clinic_settings
    ADD COLUMN IF NOT EXISTS template_id text NOT NULL DEFAULT 'klinik-minimal',
    ADD COLUMN IF NOT EXISTS show_motifs boolean NOT NULL DEFAULT true;

ALTER TABLE clinic_settings
    DROP CONSTRAINT IF EXISTS clinic_settings_template_id_check;

ALTER TABLE clinic_settings
    ADD CONSTRAINT clinic_settings_template_id_check
    CHECK (template_id IN (
        'klinik-minimal',
        'modern-geometric',
        'botanical-serenity',
        'liquid-art',
        'noir-gold',
        'royal-gold',
        'classic-ribbon',
        'executive-split',
        'botanical-aesthetic',
        'pediatric-joy',
        'prestige-minimal'
    ));
