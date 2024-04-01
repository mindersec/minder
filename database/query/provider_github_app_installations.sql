-- name: GetInstallationIDByProviderID :one
SELECT * FROM provider_github_app_installations WHERE provider_id = $1;

-- name: UpsertInstallationID :one
INSERT INTO provider_github_app_installations
    (provider_id, app_installation_id, organization_id, enrolling_user_id, enrollment_nonce, project_id)
VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (provider_id)
    DO
UPDATE SET
    app_installation_id = $2,
    organization_id = $3,
    enrolling_user_id = $4,
    enrollment_nonce = $5,
    project_id = $6,
    updated_at = NOW()
WHERE provider_github_app_installations.provider_id = $1
    RETURNING *;

-- name: GetInstallationIDByEnrollmentNonce :one
SELECT * FROM provider_github_app_installations WHERE project_id = $1 AND enrollment_nonce = $2;