-- name: CreateAccessToken :one
INSERT INTO access_tokens (organisation_id, encrypted_token) VALUES ($1, $2) RETURNING *;

-- name: GetAccessTokenByOrganisationID :one
SELECT * FROM access_tokens WHERE organisation_id = $1;

-- name: UpdateAccessToken :one
UPDATE access_tokens SET encrypted_token = $2, updated_at = NOW() WHERE organisation_id = $1 RETURNING *;

-- name: DeleteAccessToken :exec
DELETE FROM access_tokens WHERE organisation_id = $1;