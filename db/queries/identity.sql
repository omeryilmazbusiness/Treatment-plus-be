-- name: IdentityRegisterClinic :one
SELECT * FROM identity_register_clinic(sqlc.arg('id'), sqlc.arg('name'), sqlc.arg('email'), sqlc.arg('password_hash'));

-- name: IdentityFindClinicByEmail :one
SELECT * FROM identity_find_clinic_by_email(sqlc.arg('email'));

-- name: IdentityGetClinicByIDAuth :one
SELECT * FROM identity_get_clinic_by_id(sqlc.arg('id'));

-- name: IdentityGetClinicByID :one
SELECT id, name, email, password_hash, created_at, updated_at
FROM clinics
WHERE id = sqlc.arg('id');

-- name: IdentitySaveSession :one
SELECT * FROM identity_save_session(
    sqlc.arg('id'),
    sqlc.arg('clinic_id'),
    sqlc.arg('refresh_hash'),
    sqlc.arg('family_id'),
    sqlc.arg('expires_at')
);

-- name: IdentityGetSessionByRefreshHash :one
SELECT * FROM identity_get_session_by_refresh_hash(sqlc.arg('refresh_hash'));

-- name: IdentityMarkSessionReused :exec
SELECT identity_mark_session_reused(sqlc.arg('id'), sqlc.arg('reused_at'));

-- name: IdentityRevokeFamily :exec
SELECT identity_revoke_family(sqlc.arg('family_id'), sqlc.arg('revoked_at'));

-- name: IdentityCountVisibleClinics :one
SELECT count(*)::bigint AS n FROM clinics;

-- name: IdentityVisibleClinicID :one
SELECT id FROM clinics WHERE id = sqlc.arg('id');
