DROP FUNCTION IF EXISTS identity_revoke_family(uuid, timestamptz);
DROP FUNCTION IF EXISTS identity_mark_session_reused(uuid, timestamptz);
DROP FUNCTION IF EXISTS identity_get_session_by_refresh_hash(bytea);
DROP FUNCTION IF EXISTS identity_save_session(uuid, uuid, bytea, uuid, timestamptz);
DROP FUNCTION IF EXISTS identity_get_clinic_by_id(uuid);
DROP FUNCTION IF EXISTS identity_find_clinic_by_email(citext);
DROP FUNCTION IF EXISTS identity_register_clinic(uuid, text, citext, text);

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS clinics;

DROP FUNCTION IF EXISTS set_updated_at();
