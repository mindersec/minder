-- name: GetInstallationIDByProviderID :one
SELECT * FROM provider_github_app_installations WHERE provider_id = $1;

-- name: GetInstallationIDByAppID :one
SELECT * FROM provider_github_app_installations WHERE app_installation_id = $1;

-- name: UpsertInstallationID :one
INSERT INTO provider_github_app_installations
    (organization_id, app_installation_id, provider_id, enrolling_user_id, enrollment_nonce, project_id, is_org)
VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (organization_id)
    DO
UPDATE SET
    app_installation_id = $2,
    provider_id = $3,
    enrolling_user_id = $4,
    enrollment_nonce = $5,
    project_id = $6,
    is_org = $7,
    updated_at = NOW()
WHERE provider_github_app_installations.organization_id = $1
    RETURNING *;

-- name: GetUnclaimedInstallationsByUser :many
SELECT * FROM provider_github_app_installations WHERE enrolling_user_id = sqlc.arg('gh_id') AND provider_id IS NULL;

-- name: GetInstallationIDByEnrollmentNonce :one
SELECT * FROM provider_github_app_installations WHERE project_id = $1 AND enrollment_nonce = $2;

-- name: DeleteInstallationIDByAppID :exec
DELETE FROM provider_github_app_installations WHERE app_installation_id = $1;