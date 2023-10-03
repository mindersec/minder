-- name: CreateSigningKey :one
INSERT INTO signing_keys (project_id, private_key, public_key, passphrase, key_identifier, created_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetSigningKeyByProjectID :one
SELECT * FROM signing_keys WHERE project_id = $1;

-- name: DeleteSigningKey :exec
DELETE FROM signing_keys WHERE project_id = $1 AND key_identifier = $2;

-- name: GetSigningKeyByIdentifier :one
SELECT * FROM signing_keys WHERE key_identifier = $1;