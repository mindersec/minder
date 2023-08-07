-- name: CreateAccessToken :one
INSERT INTO provider_access_tokens (group_id, provider, encrypted_token, expiration_time, owner_filter) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetAccessTokenByGroupID :one
SELECT * FROM provider_access_tokens WHERE provider = $1 AND group_id = $2;

-- name: UpdateAccessToken :one
UPDATE provider_access_tokens SET encrypted_token = $3, expiration_time = $4, owner_filter = $5, updated_at = NOW() WHERE provider = $1 AND group_id = $2 RETURNING *;

-- name: DeleteAccessToken :exec
DELETE FROM provider_access_tokens WHERE provider = $1 AND group_id = $2;

-- name: GetAccessTokenByProvider :many
SELECT * FROM provider_access_tokens WHERE provider = $1;

-- name: GetAccessTokenSinceDate :one
SELECT * FROM provider_access_tokens WHERE provider = $1 AND group_id = $2 AND created_at >= $3;
