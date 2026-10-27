DROP FUNCTION IF EXISTS identity_save_session(uuid, uuid, uuid, identity_role, bytea, uuid, timestamptz);
DROP FUNCTION IF EXISTS identity_find_account_by_workspace(uuid);
DROP FUNCTION IF EXISTS identity_find_account_by_email(citext);
DROP FUNCTION IF EXISTS identity_register_account(uuid, uuid, text, citext, text, identity_role);
DROP TABLE IF EXISTS identity_account_row;

ALTER TABLE sessions DROP COLUMN IF EXISTS role;
ALTER TABLE sessions DROP COLUMN IF EXISTS user_id;

DROP TABLE IF EXISTS memberships;

ALTER TABLE clinics ADD COLUMN email citext;
ALTER TABLE clinics ADD COLUMN password_hash text;
UPDATE clinics c
SET email = u.email, password_hash = u.password_hash
FROM users u
WHERE u.id = c.owner_user_id;
ALTER TABLE clinics DROP COLUMN IF EXISTS kind;
ALTER TABLE clinics DROP COLUMN IF EXISTS owner_user_id;

DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS identity_role;
