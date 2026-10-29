-- Phase: Ultimate presentation, share links, patient analytics & digital acceptance

-- 1. Extend plans (support ultimate/both, acceptance, welcome letter, video, before/after cases)
ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_kind_check;
ALTER TABLE plans ADD CONSTRAINT plans_kind_check CHECK (kind IN ('classic', 'ultimate', 'both'));

ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_status_check;
ALTER TABLE plans ADD CONSTRAINT plans_status_check CHECK (status IN ('draft', 'ready', 'sent', 'viewed', 'accepted', 'expired'));

ALTER TABLE plans
    ADD COLUMN welcome_letter text NOT NULL DEFAULT '',
    ADD COLUMN video_url text NOT NULL DEFAULT '',
    ADD COLUMN before_after_case_ids text[] NOT NULL DEFAULT '{}';

-- 2. Extend clinic_settings (team, reviews, before_after, cta)
ALTER TABLE clinic_settings
    ADD COLUMN team_json jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN reviews_json jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN before_after_json jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN cta_json jsonb NOT NULL DEFAULT '{}'::jsonb;

-- 3. Share links (tokenized public access to plans)
CREATE TABLE share_links (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    plan_id uuid NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    token_prefix text NOT NULL DEFAULT '',
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    views_count integer NOT NULL DEFAULT 0,
    last_viewed_at timestamptz,
    accepted_at timestamptz,
    accepted_signature text NOT NULL DEFAULT '',
    accepted_ip text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX share_links_workspace_idx ON share_links (workspace_id);
CREATE INDEX share_links_plan_idx ON share_links (plan_id);
CREATE INDEX share_links_token_hash_idx ON share_links (token_hash);

CREATE TRIGGER share_links_set_updated_at
    BEFORE UPDATE ON share_links
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE share_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE share_links FORCE ROW LEVEL SECURITY;

CREATE POLICY share_links_tenant ON share_links
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 4. Share events (patient engagement analytics)
CREATE TABLE share_events (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    link_id uuid NOT NULL REFERENCES share_links (id) ON DELETE CASCADE,
    kind text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX share_events_workspace_idx ON share_events (workspace_id, created_at DESC);
CREATE INDEX share_events_link_idx ON share_events (link_id, created_at DESC);

ALTER TABLE share_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE share_events FORCE ROW LEVEL SECURITY;

CREATE POLICY share_events_tenant ON share_events
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON share_links, share_events TO treatment_app;

-- 5. Public helper functions (SECURITY DEFINER)
CREATE OR REPLACE FUNCTION share_resolve_token(p_token_hash bytea)
RETURNS TABLE (
    link_id uuid,
    workspace_id uuid,
    plan_id uuid,
    expires_at timestamptz,
    revoked_at timestamptz,
    views_count integer,
    last_viewed_at timestamptz,
    accepted_at timestamptz,
    accepted_signature text
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
SET row_security = off
AS $$
DECLARE
    rec share_links;
BEGIN
    SELECT * INTO rec
    FROM share_links
    WHERE token_hash = p_token_hash;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    IF rec.revoked_at IS NOT NULL OR rec.expires_at < now() THEN
        link_id := rec.id;
        workspace_id := rec.workspace_id;
        plan_id := rec.plan_id;
        expires_at := rec.expires_at;
        revoked_at := rec.revoked_at;
        views_count := rec.views_count;
        last_viewed_at := rec.last_viewed_at;
        accepted_at := rec.accepted_at;
        accepted_signature := rec.accepted_signature;
        RETURN NEXT;
        RETURN;
    END IF;

    UPDATE share_links AS sl
    SET views_count = sl.views_count + 1,
        last_viewed_at = now(),
        updated_at = now()
    WHERE sl.id = rec.id
    RETURNING * INTO rec;

    UPDATE plans
    SET status = 'viewed',
        updated_at = now()
    WHERE id = rec.plan_id AND status IN ('ready', 'sent');

    link_id := rec.id;
    workspace_id := rec.workspace_id;
    plan_id := rec.plan_id;
    expires_at := rec.expires_at;
    revoked_at := rec.revoked_at;
    views_count := rec.views_count;
    last_viewed_at := rec.last_viewed_at;
    accepted_at := rec.accepted_at;
    accepted_signature := rec.accepted_signature;
    RETURN NEXT;
END;
$$;

CREATE OR REPLACE FUNCTION share_record_event(
    p_token_hash bytea,
    p_event_id uuid,
    p_kind text,
    p_metadata jsonb
)
RETURNS boolean
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
SET row_security = off
AS $$
DECLARE
    v_link_id uuid;
    v_workspace_id uuid;
BEGIN
    SELECT id, workspace_id INTO v_link_id, v_workspace_id
    FROM share_links
    WHERE token_hash = p_token_hash
      AND revoked_at IS NULL
      AND expires_at >= now();

    IF NOT FOUND THEN
        RETURN false;
    END IF;

    INSERT INTO share_events (id, workspace_id, link_id, kind, metadata, created_at)
    VALUES (p_event_id, v_workspace_id, v_link_id, p_kind, COALESCE(p_metadata, '{}'::jsonb), now());

    RETURN true;
END;
$$;

CREATE OR REPLACE FUNCTION share_accept_plan(
    p_token_hash bytea,
    p_signature text,
    p_ip text
)
RETURNS boolean
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
SET row_security = off
AS $$
DECLARE
    v_link_id uuid;
    v_plan_id uuid;
    v_workspace_id uuid;
BEGIN
    SELECT id, plan_id, workspace_id INTO v_link_id, v_plan_id, v_workspace_id
    FROM share_links
    WHERE token_hash = p_token_hash
      AND revoked_at IS NULL
      AND expires_at >= now();

    IF NOT FOUND THEN
        RETURN false;
    END IF;

    UPDATE share_links
    SET accepted_at = now(),
        accepted_signature = COALESCE(p_signature, ''),
        accepted_ip = COALESCE(p_ip, ''),
        updated_at = now()
    WHERE id = v_link_id;

    UPDATE plans
    SET status = 'accepted',
        updated_at = now()
    WHERE id = v_plan_id;

    INSERT INTO share_events (id, workspace_id, link_id, kind, metadata, created_at)
    VALUES (
        gen_random_uuid(),
        v_workspace_id,
        v_link_id,
        'accepted',
        jsonb_build_object('signature', p_signature, 'ip', p_ip),
        now()
    );

    RETURN true;
END;
$$;

GRANT EXECUTE ON FUNCTION share_resolve_token, share_record_event, share_accept_plan TO treatment_app;
