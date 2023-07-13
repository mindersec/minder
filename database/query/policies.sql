-- name: CreatePolicy :one
INSERT INTO policies (  
    provider,
    group_id,
    policy_type,
    policy_definition) VALUES ($1, $2, $3, sqlc.arg(policy_definition)::jsonb) RETURNING *;

-- name: GetPolicyByID :one
SELECT policies.id as id, policies.provider as provider, group_id, policies.policy_type as policy_type,
policy_definition, policy_types.policy_type as policy_type_name,
policies.created_at as created_at, policies.updated_at as updated_at FROM policies
LEFT OUTER JOIN policy_types ON policy_types.id = policies.policy_type WHERE policies.id = $1;

-- name: ListPoliciesByGroupID :many
SELECT policies.id as id, policies.provider as provider, group_id, policies.policy_type as policy_type,
policy_definition, policy_types.policy_type as policy_type_name,
policies.created_at as created_at, policies.updated_at as updated_at FROM policies
LEFT OUTER JOIN policy_types ON policy_types.id = policies.policy_type
WHERE policies.provider = $1 AND group_id = $2
ORDER BY policies.id
LIMIT $3
OFFSET $4;

-- name: DeletePolicy :exec
DELETE FROM policies
WHERE id = $1;

-- name: GetPoliciesByRepoAndType :many
SELECT p.id, p.provider, p.group_id, p.policy_type, p.policy_definition FROM repositories r
INNER JOIN policies p ON p.provider=r.provider AND p.group_id=r.group_id AND
p.policy_type=(SELECT id FROM policy_types pt WHERE pt.provider=p.provider AND pt.policy_type=$2)
WHERE r.provider = $1 AND r.repo_id = $3;

