DROP FUNCTION IF EXISTS share_accept_plan(bytea, text, text);
DROP FUNCTION IF EXISTS share_record_event(bytea, uuid, text, jsonb);
DROP FUNCTION IF EXISTS share_resolve_token(bytea);

DROP TABLE IF EXISTS share_events;
DROP TABLE IF EXISTS share_links;

ALTER TABLE clinic_settings
    DROP COLUMN IF EXISTS team_json,
    DROP COLUMN IF EXISTS reviews_json,
    DROP COLUMN IF EXISTS before_after_json,
    DROP COLUMN IF EXISTS cta_json;

ALTER TABLE plans
    DROP COLUMN IF EXISTS welcome_letter,
    DROP COLUMN IF EXISTS video_url,
    DROP COLUMN IF EXISTS before_after_case_ids;

ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_kind_check;
ALTER TABLE plans ADD CONSTRAINT plans_kind_check CHECK (kind = 'classic');

ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_status_check;
ALTER TABLE plans ADD CONSTRAINT plans_status_check CHECK (status IN ('draft', 'ready'));

