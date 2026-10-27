-- Task 1: clinic is the only actor. No users, memberships, or roles.
CREATE TABLE clinics (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    email citext NOT NULL,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT clinics_email_key UNIQUE (email),
    CONSTRAINT clinics_name_len CHECK (char_length(name) BETWEEN 2 AND 120),
    CONSTRAINT clinics_email_len CHECK (char_length(email::text) BETWEEN 3 AND 254),
    CONSTRAINT clinics_password_hash_len CHECK (char_length(password_hash) >= 20)
);

CREATE TABLE sessions (
    id uuid PRIMARY KEY,
    clinic_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    refresh_hash bytea NOT NULL,
    family_id uuid NOT NULL,
    reused_at timestamptz,
    revoked_at timestamptz,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sessions_refresh_hash_key UNIQUE (refresh_hash),
    CONSTRAINT sessions_refresh_hash_len CHECK (octet_length(refresh_hash) = 32)
);

CREATE INDEX sessions_clinic_id_idx ON sessions (clinic_id);
CREATE INDEX sessions_family_id_idx ON sessions (family_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER clinics_set_updated_at
    BEFORE UPDATE ON clinics
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE clinics ENABLE ROW LEVEL SECURITY;
ALTER TABLE clinics FORCE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions FORCE ROW LEVEL SECURITY;

-- Tenant isolation. Missing app.clinic_id → 0 rows (FORCE applies to table owner too).
CREATE POLICY clinics_tenant ON clinics
    USING (id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

CREATE POLICY sessions_tenant ON sessions
    USING (clinic_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (clinic_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- Auth bootstrap runs as owner with row_security off. App SELECTs stay under RLS.
CREATE OR REPLACE FUNCTION identity_register_clinic(
    p_id uuid,
    p_name text,
    p_email citext,
    p_password_hash text
)
RETURNS clinics
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
SET row_security = off
AS $$
DECLARE
    rec clinics;
BEGIN
    INSERT INTO clinics (id, name, email, password_hash)
    VALUES (p_id, p_name, p_email, p_password_hash)
    RETURNING * INTO rec;
    RETURN rec;
END;
$$;

CREATE OR REPLACE FUNCTION identity_find_clinic_by_email(p_email citext)
RETURNS SETOF clinics
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
SET row_security = off
STABLE
AS $$
    SELECT * FROM clinics WHERE email = p_email;
$$;

CREATE OR REPLACE FUNCTION identity_get_clinic_by_id(p_id uuid)
RETURNS SETOF clinics
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
SET row_security = off
STABLE
AS $$
    SELECT * FROM clinics WHERE id = p_id;
$$;

CREATE OR REPLACE FUNCTION identity_save_session(
    p_id uuid,
    p_clinic_id uuid,
    p_refresh_hash bytea,
    p_family_id uuid,
    p_expires_at timestamptz
)
RETURNS sessions
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
SET row_security = off
AS $$
DECLARE
    rec sessions;
BEGIN
    INSERT INTO sessions (id, clinic_id, refresh_hash, family_id, expires_at)
    VALUES (p_id, p_clinic_id, p_refresh_hash, p_family_id, p_expires_at)
    RETURNING * INTO rec;
    RETURN rec;
END;
$$;

CREATE OR REPLACE FUNCTION identity_get_session_by_refresh_hash(p_hash bytea)
RETURNS SETOF sessions
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
SET row_security = off
STABLE
AS $$
    SELECT * FROM sessions WHERE refresh_hash = p_hash;
$$;

CREATE OR REPLACE FUNCTION identity_mark_session_reused(p_id uuid, p_at timestamptz)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
SET row_security = off
AS $$
BEGIN
    UPDATE sessions
    SET reused_at = p_at
    WHERE id = p_id
      AND reused_at IS NULL;
END;
$$;

CREATE OR REPLACE FUNCTION identity_revoke_family(p_family_id uuid, p_at timestamptz)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
SET row_security = off
AS $$
BEGIN
    UPDATE sessions
    SET revoked_at = COALESCE(revoked_at, p_at)
    WHERE family_id = p_family_id
      AND revoked_at IS NULL;
END;
$$;

REVOKE ALL ON FUNCTION identity_register_clinic(uuid, text, citext, text) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_find_clinic_by_email(citext) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_get_clinic_by_id(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_save_session(uuid, uuid, bytea, uuid, timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_get_session_by_refresh_hash(bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_mark_session_reused(uuid, timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_revoke_family(uuid, timestamptz) FROM PUBLIC;

GRANT EXECUTE ON FUNCTION identity_register_clinic(uuid, text, citext, text) TO CURRENT_USER;
GRANT EXECUTE ON FUNCTION identity_find_clinic_by_email(citext) TO CURRENT_USER;
GRANT EXECUTE ON FUNCTION identity_get_clinic_by_id(uuid) TO CURRENT_USER;
GRANT EXECUTE ON FUNCTION identity_save_session(uuid, uuid, bytea, uuid, timestamptz) TO CURRENT_USER;
GRANT EXECUTE ON FUNCTION identity_get_session_by_refresh_hash(bytea) TO CURRENT_USER;
GRANT EXECUTE ON FUNCTION identity_mark_session_reused(uuid, timestamptz) TO CURRENT_USER;
GRANT EXECUTE ON FUNCTION identity_revoke_family(uuid, timestamptz) TO CURRENT_USER;
