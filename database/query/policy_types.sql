-- name: CreatePolicyType :one
INSERT INTO policy_types (
  provider,
  policy_type,
  description,
  json_schema,
  version) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetPolicyTypeById :one
SELECT id, policy_type, description, json_schema, version, created_at, updated_at FROM policy_types WHERE id = $1;

-- name: GetPolicyType :one
SELECT id, policy_type, description, json_schema, version, created_at, updated_at FROM policy_types WHERE provider = $1 AND policy_type = $1;

-- name: GetPolicyTypes :many
SELECT id, provider, policy_type, description, json_schema, version, created_at, updated_at FROM policy_types WHERE provider = $1 ORDER BY policy_type;

-- name: DeletePolicyType :exec
DELETE FROM policy_types WHERE policy_type = $1;