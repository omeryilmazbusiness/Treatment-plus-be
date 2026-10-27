-- name: IdentityRegisterAccount :exec
SELECT identity_register_account(
    sqlc.arg('user_id'),
    sqlc.arg('workspace_id'),
    sqlc.arg('name'),
    sqlc.arg('email'),
    sqlc.arg('password_hash'),
    sqlc.arg('role')
);

-- name: IdentityFindAccountByEmail :one
SELECT user_id, email, password_hash, user_created_at, workspace_id, workspace_name, workspace_kind, role
FROM identity_find_account_by_email(sqlc.arg('email'));

-- name: IdentityFindAccountByWorkspace :one
SELECT user_id, email, password_hash, user_created_at, workspace_id, workspace_name, workspace_kind, role
FROM identity_find_account_by_workspace(sqlc.arg('workspace_id'));

-- name: IdentityGetClinicByIDAuth :one
SELECT * FROM identity_get_clinic_by_id(sqlc.arg('id'));

-- name: IdentityGetClinicByID :one
SELECT id, name, owner_user_id, kind, created_at, updated_at
FROM clinics
WHERE id = sqlc.arg('id');

-- name: IdentitySaveSession :one
SELECT * FROM identity_save_session(
    sqlc.arg('id'),
    sqlc.arg('user_id'),
    sqlc.arg('clinic_id'),
    sqlc.arg('role'),
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
