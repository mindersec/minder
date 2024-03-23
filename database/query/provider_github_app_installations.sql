-- name: GetInstallationIDByProviderID :one
SELECT * FROM provider_github_app_installations WHERE provider_id = $1;