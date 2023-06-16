-- name: CreateAccessToken :one
INSERT INTO provider_access_tokens (group_id, provider, encrypted_token, expiration_time) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetAccessTokenByGroupID :one
SELECT * FROM provider_access_tokens WHERE provider = $1 AND group_id = $2;

-- name: UpdateAccessToken :one
UPDATE provider_access_tokens SET provider = $2, encrypted_token = $3, expiration_time = $4, updated_at = NOW() WHERE provider = $1 AND group_id = $2 RETURNING *;

-- name: DeleteAccessToken :exec
DELETE FROM provider_access_tokens WHERE provider = $1 AND group_id = $2;

-- name: GetAccessTokenByProvider :many
SELECT * FROM provider_access_tokens WHERE provider = $1;