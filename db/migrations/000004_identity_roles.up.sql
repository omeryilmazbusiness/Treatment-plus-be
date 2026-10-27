-- User logs in. Role (clinic|personal) authorizes. Workspace (clinics) is the tenant.

CREATE TYPE identity_role AS ENUM ('clinic', 'personal');

CREATE TABLE users (
    id uuid PRIMARY KEY,
    email citext NOT NULL,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_email_len CHECK (char_length(email::text) BETWEEN 3 AND 254),
    CONSTRAINT users_password_hash_len CHECK (char_length(password_hash) >= 20)
);

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

INSERT INTO users (id, email, password_hash, created_at, updated_at)
SELECT id, email, password_hash, created_at, updated_at FROM clinics;

ALTER TABLE clinics ADD COLUMN owner_user_id uuid REFERENCES users (id);
ALTER TABLE clinics ADD COLUMN kind identity_role NOT NULL DEFAULT 'clinic';
UPDATE clinics SET owner_user_id = id WHERE owner_user_id IS NULL;
ALTER TABLE clinics ALTER COLUMN owner_user_id SET NOT NULL;

ALTER TABLE clinics DROP CONSTRAINT clinics_email_key;
ALTER TABLE clinics DROP CONSTRAINT clinics_email_len;
ALTER TABLE clinics DROP CONSTRAINT clinics_password_hash_len;
ALTER TABLE clinics DROP COLUMN email;
ALTER TABLE clinics DROP COLUMN password_hash;

CREATE TABLE memberships (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    role identity_role NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, workspace_id)
);

INSERT INTO memberships (user_id, workspace_id, role)
SELECT owner_user_id, id, kind FROM clinics;

CREATE INDEX memberships_workspace_id_idx ON memberships (workspace_id);

ALTER TABLE sessions ADD COLUMN user_id uuid REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE sessions ADD COLUMN role identity_role;
UPDATE sessions SET user_id = clinic_id, role = 'clinic' WHERE user_id IS NULL;
ALTER TABLE sessions ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE sessions ALTER COLUMN role SET NOT NULL;

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships FORCE ROW LEVEL SECURITY;

CREATE POLICY users_self ON users
    USING (id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE POLICY memberships_tenant ON memberships
    USING (
        workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid
        OR user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
    );

GRANT SELECT, INSERT, UPDATE, DELETE ON users, memberships TO treatment_app;

DROP FUNCTION IF EXISTS identity_register_clinic(uuid, text, citext, text);
DROP FUNCTION IF EXISTS identity_find_clinic_by_email(citext);
DROP FUNCTION IF EXISTS identity_save_session(uuid, uuid, bytea, uuid, timestamptz);

CREATE OR REPLACE FUNCTION identity_register_account(
    p_user_id uuid,
    p_workspace_id uuid,
    p_name text,
    p_email citext,
    p_password_hash text,
    p_role identity_role
)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
SET row_security = off
AS $$
BEGIN
    INSERT INTO users (id, email, password_hash)
    VALUES (p_user_id, p_email, p_password_hash);
    INSERT INTO clinics (id, name, owner_user_id, kind)
    VALUES (p_workspace_id, p_name, p_user_id, p_role);
    INSERT INTO memberships (user_id, workspace_id, role)
    VALUES (p_user_id, p_workspace_id, p_role);
END;
$$;

CREATE TABLE identity_account_row (
    user_id uuid NOT NULL,
    email citext NOT NULL,
    password_hash text NOT NULL,
    user_created_at timestamptz NOT NULL,
    workspace_id uuid NOT NULL,
    workspace_name text NOT NULL,
    workspace_kind identity_role NOT NULL,
    role identity_role NOT NULL
);

CREATE OR REPLACE FUNCTION identity_find_account_by_email(p_email citext)
RETURNS SETOF identity_account_row
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
SET row_security = off
STABLE
AS $$
    SELECT u.id, u.email, u.password_hash, u.created_at,
           w.id, w.name, w.kind, m.role
    FROM users u
    JOIN memberships m ON m.user_id = u.id
    JOIN clinics w ON w.id = m.workspace_id
    WHERE u.email = p_email
    LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION identity_find_account_by_workspace(p_workspace_id uuid)
RETURNS SETOF identity_account_row
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
SET row_security = off
STABLE
AS $$
    SELECT u.id, u.email, u.password_hash, u.created_at,
           w.id, w.name, w.kind, m.role
    FROM clinics w
    JOIN memberships m ON m.workspace_id = w.id
    JOIN users u ON u.id = m.user_id
    WHERE w.id = p_workspace_id
    LIMIT 1;
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
    p_user_id uuid,
    p_clinic_id uuid,
    p_role identity_role,
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
    INSERT INTO sessions (id, user_id, clinic_id, role, refresh_hash, family_id, expires_at)
    VALUES (p_id, p_user_id, p_clinic_id, p_role, p_refresh_hash, p_family_id, p_expires_at)
    RETURNING * INTO rec;
    RETURN rec;
END;
$$;

REVOKE ALL ON FUNCTION identity_register_account(uuid, uuid, text, citext, text, identity_role) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_find_account_by_email(citext) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_find_account_by_workspace(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_get_clinic_by_id(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION identity_save_session(uuid, uuid, uuid, identity_role, bytea, uuid, timestamptz) FROM PUBLIC;

GRANT EXECUTE ON FUNCTION identity_register_account(uuid, uuid, text, citext, text, identity_role) TO CURRENT_USER, treatment_app;
GRANT EXECUTE ON FUNCTION identity_find_account_by_email(citext) TO CURRENT_USER, treatment_app;
GRANT EXECUTE ON FUNCTION identity_find_account_by_workspace(uuid) TO CURRENT_USER, treatment_app;
GRANT EXECUTE ON FUNCTION identity_get_clinic_by_id(uuid) TO CURRENT_USER, treatment_app;
GRANT EXECUTE ON FUNCTION identity_save_session(uuid, uuid, uuid, identity_role, bytea, uuid, timestamptz) TO CURRENT_USER, treatment_app;
GRANT EXECUTE ON FUNCTION identity_get_session_by_refresh_hash(bytea) TO CURRENT_USER, treatment_app;
GRANT EXECUTE ON FUNCTION identity_mark_session_reused(uuid, timestamptz) TO CURRENT_USER, treatment_app;
GRANT EXECUTE ON FUNCTION identity_revoke_family(uuid, timestamptz) TO CURRENT_USER, treatment_app;
