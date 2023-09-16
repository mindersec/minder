-- name: CreateAccessToken :one
INSERT INTO provider_access_tokens (provider_id, encrypted_token, expiration_time, owner_filter) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetAccessTokenByProviderID :one
SELECT * FROM provider_access_tokens WHERE provider_id = $1;

-- name: UpdateAccessToken :one
UPDATE provider_access_tokens SET encrypted_token = $2, expiration_time = $3, owner_filter = $4, updated_at = NOW() WHERE provider_id = $1 RETURNING *;

-- name: DeleteAccessToken :exec
DELETE FROM provider_access_tokens WHERE provider_id = $1;

-- name: GetAccessTokenByProvider :many
SELECT * FROM provider_access_tokens WHERE provider_id = $1;

-- name: GetAccessTokenSinceDate :one
SELECT * FROM provider_access_tokens WHERE provider_id = $1 AND created_at >= $2;
