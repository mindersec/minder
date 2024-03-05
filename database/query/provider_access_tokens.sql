-- name: GetAccessTokenByProjectID :one
SELECT * FROM provider_access_tokens WHERE provider = $1 AND project_id = $2;

-- name: GetAccessTokenByProvider :many
SELECT * FROM provider_access_tokens WHERE provider = $1;

-- name: GetAccessTokenSinceDate :one
SELECT * FROM provider_access_tokens WHERE provider = $1 AND project_id = $2 AND updated_at >= $3;

-- name: UpsertAccessToken :one
INSERT INTO provider_access_tokens
(project_id, provider, encrypted_token, expiration_time, owner_filter)
VALUES
    ($1, $2, $3, $4, $5)
ON CONFLICT (project_id, provider)
    DO UPDATE SET
                  encrypted_token = $3,
                  expiration_time = $4,
                  owner_filter = $5,
                  updated_at = NOW()
WHERE provider_access_tokens.project_id = $1 AND provider_access_tokens.provider = $2
RETURNING *;