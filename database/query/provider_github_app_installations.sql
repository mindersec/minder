-- name: GetInstallationIDByProviderID :one
SELECT * FROM provider_github_app_installations WHERE provider_id = $1;

-- name: UpsertInstallationID :one
INSERT INTO provider_github_app_installations
    (provider_id, app_installation_id, organization_id, enrolling_user_id)
VALUES ($1, $2, $3, $4) ON CONFLICT (provider_id)
    DO
UPDATE SET
    app_installation_id = $2,
    organization_id = $3,
    enrolling_user_id = $4,
    updated_at = NOW()
WHERE provider_github_app_installations.provider_id = $1
    RETURNING *;