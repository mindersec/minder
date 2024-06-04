-- name: GetAccessTokenByProjectID :one
SELECT * FROM provider_access_tokens WHERE provider = $1 AND project_id = $2;

-- name: GetAccessTokenByProvider :many
SELECT * FROM provider_access_tokens WHERE provider = $1;

-- name: GetAccessTokenSinceDate :one
SELECT * FROM provider_access_tokens WHERE provider = $1 AND project_id = $2 AND updated_at >= $3;

-- name: UpsertAccessToken :one
INSERT INTO provider_access_tokens
(project_id, provider, expiration_time, owner_filter, enrollment_nonce, encrypted_access_token)
VALUES
    ($1, $2, $3, $4, $5, $6)
ON CONFLICT (project_id, provider)
    DO UPDATE SET
                  expiration_time = $3,
                  owner_filter = $4,
                  enrollment_nonce = $5,
                  updated_at = NOW(),
                  encrypted_access_token = $6
WHERE provider_access_tokens.project_id = $1 AND provider_access_tokens.provider = $2
RETURNING *;

-- name: GetAccessTokenByEnrollmentNonce :one
SELECT * FROM provider_access_tokens WHERE project_id = $1 AND enrollment_nonce = $2;

-- When doing a key/algorithm rotation, identify the secrets which need to be
-- rotated. The criteria for rotation are:
-- 1) The encrypted_access_token is NULL (this should be removed when we make
--    this column non-nullable).
-- 2) The access token does not use the configured default algorithm.
-- 3) The access token does not use the default key version.
-- This query accepts the default key version/algorithm as arguments since
-- that information is not known to the database.
-- name: ListTokensToMigrate :many
SELECT * FROM provider_access_tokens WHERE
    encrypted_access_token IS NULL OR
    encrypted_access_token->>'Algorithm'  <> sqlc.arg(default_algorithm)::TEXT OR
    encrypted_access_token->>'KeyVersion' <> sqlc.arg(default_key_version)::TEXT
LIMIT  sqlc.arg(batch_size)::bigint
OFFSET sqlc.arg(batch_offset)::bigint;

-- name: UpdateEncryptedSecret :exec
UPDATE provider_access_tokens
SET encrypted_access_token = sqlc.arg(secret)::JSONB
WHERE id = $1;