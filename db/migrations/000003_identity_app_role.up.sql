-- App role is not a superuser. Superusers bypass FORCE ROW LEVEL SECURITY.
-- Migrate keeps using the owner (treatment). The API uses treatment_app.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'treatment_app') THEN
        CREATE ROLE treatment_app LOGIN PASSWORD 'treatment'
            NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
    END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO treatment_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON clinics, sessions TO treatment_app;

GRANT EXECUTE ON FUNCTION identity_register_clinic(uuid, text, citext, text) TO treatment_app;
GRANT EXECUTE ON FUNCTION identity_find_clinic_by_email(citext) TO treatment_app;
GRANT EXECUTE ON FUNCTION identity_get_clinic_by_id(uuid) TO treatment_app;
GRANT EXECUTE ON FUNCTION identity_save_session(uuid, uuid, bytea, uuid, timestamptz) TO treatment_app;
GRANT EXECUTE ON FUNCTION identity_get_session_by_refresh_hash(bytea) TO treatment_app;
GRANT EXECUTE ON FUNCTION identity_mark_session_reused(uuid, timestamptz) TO treatment_app;
GRANT EXECUTE ON FUNCTION identity_revoke_family(uuid, timestamptz) TO treatment_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO treatment_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO treatment_app;
