-- name: CreateAccessToken :one
INSERT INTO provider_access_tokens (group_id, encrypted_token) VALUES ($1, $2) RETURNING *;

-- name: GetAccessTokenByGroupID :one
SELECT * FROM provider_access_tokens WHERE group_id = $1;

-- name: UpdateAccessToken :one
UPDATE provider_access_tokens SET encrypted_token = $2, updated_at = NOW() WHERE group_id = $1 RETURNING *;

-- name: DeleteAccessToken :exec
DELETE FROM provider_access_tokens WHERE group_id = $1;