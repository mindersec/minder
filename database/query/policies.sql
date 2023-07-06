-- name: CreatePolicy :one
INSERT INTO policies (  
    provider,
    group_id,
    policy_type,
    policy_definition) VALUES ($1, $2, $3, sqlc.arg(policy_definition)::jsonb) RETURNING *;

-- name: GetPolicyByID :one
SELECT id, provider, group_id, policy_type, policy_definition, created_at, updated_at FROM policies WHERE id = $1;

-- name: ListPoliciesByGroupID :many
SELECT id, provider, group_id, policy_type, policy_definition, created_at, updated_at FROM policies
WHERE provider = $1 AND group_id = $2
ORDER BY id
LIMIT $3
OFFSET $4;

-- name: DeletePolicy :exec
DELETE FROM policies
WHERE id = $1;