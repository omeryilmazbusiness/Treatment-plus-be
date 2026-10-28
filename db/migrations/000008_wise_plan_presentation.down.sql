ALTER TABLE plan_items
    DROP COLUMN IF EXISTS teeth,
    DROP COLUMN IF EXISTS description;

ALTER TABLE plan_visits
    DROP COLUMN IF EXISTS phase_label,
    DROP COLUMN IF EXISTS healing_note,
    DROP COLUMN IF EXISTS chart_teeth;

ALTER TABLE plans
    DROP CONSTRAINT IF EXISTS plans_palette_check,
    DROP CONSTRAINT IF EXISTS plans_locale_check;

ALTER TABLE plans
    DROP COLUMN IF EXISTS palette,
    DROP COLUMN IF EXISTS locale,
    DROP COLUMN IF EXISTS situation_json;
