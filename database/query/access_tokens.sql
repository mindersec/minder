-- name: CreateAccessToken :one
INSERT INTO user_access_tokens (user_id, encrypted_token, token_expiry,refresh_token, refresh_token_expiry) 
VALUES ($1, $2, $3, $4, $5) 
RETURNING *;

-- name: GetAccessTokenByUserID :one
SELECT id, user_id, encrypted_token, refresh_token, created_at, updated_at
FROM user_access_tokens
WHERE id = $1;

-- name: UpdateAccessToken :one
UPDATE user_access_tokens SET user_id = $1, encrypted_token = $2, token_expiry = $3, refresh_token = $4,  refresh_token_expiry = $5 WHERE id = $4 
RETURNING *;
