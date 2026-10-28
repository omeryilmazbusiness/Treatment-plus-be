-- Per-PDF presentation (palette/locale) and Wise situation + visit extras.

ALTER TABLE plans
    ADD COLUMN palette text NOT NULL DEFAULT 'navy',
    ADD COLUMN locale text NOT NULL DEFAULT 'tr',
    ADD COLUMN situation_json jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE plans
    ADD CONSTRAINT plans_palette_check CHECK (palette IN ('navy', 'forest', 'rose', 'slate')),
    ADD CONSTRAINT plans_locale_check CHECK (locale IN ('tr', 'en'));

ALTER TABLE plan_visits
    ADD COLUMN phase_label text NOT NULL DEFAULT '',
    ADD COLUMN healing_note text NOT NULL DEFAULT '',
    ADD COLUMN chart_teeth text[] NOT NULL DEFAULT '{}';

ALTER TABLE plan_items
    ADD COLUMN teeth text[] NOT NULL DEFAULT '{}',
    ADD COLUMN description text NOT NULL DEFAULT '';

UPDATE plan_items
SET teeth = ARRAY[tooth_number]
WHERE tooth_number IS NOT NULL AND tooth_number <> '' AND cardinality(teeth) = 0;
