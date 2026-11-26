ALTER TABLE clinic_settings
    DROP CONSTRAINT IF EXISTS clinic_settings_template_id_check,
    DROP COLUMN IF EXISTS show_motifs,
    DROP COLUMN IF EXISTS template_id;
