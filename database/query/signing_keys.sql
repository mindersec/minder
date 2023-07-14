-- name: CreateSigningKey :one
INSERT INTO signing_keys (group_id, private_key, public_key, passphrase, key_identifier, created_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetSigningKeyByGroupID :one
SELECT * FROM signing_keys WHERE group_id = $1;

-- name: DeleteSigningKey :exec
DELETE FROM signing_keys WHERE group_id = $1 AND key_identifier = $2;

-- name: GetSigningKeyByIdentifier :one
SELECT * FROM signing_keys WHERE key_identifier = $1;