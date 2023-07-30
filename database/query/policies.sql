-- name: CreatePolicy :one
INSERT INTO policies (  
    provider,
    group_id,
    name) VALUES ($1, $2, $3) RETURNING *;

-- name: CreatePolicyForEntity :one
INSERT INTO entity_policies (
    entity,
    policy_id,
    contextual_rules) VALUES ($1, $2, sqlc.arg(contextual_rules)::jsonb) RETURNING *;

-- name: GetPolicyByGroupAndID :many
SELECT * FROM policies JOIN entity_policies ON policies.id = entity_policies.policy_id
WHERE policies.group_id = $1 AND policies.id = $2;

-- name: GetPolicyByGroupAndName :many
SELECT * FROM policies JOIN entity_policies ON policies.id = entity_policies.policy_id
WHERE policies.group_id = $1 AND policies.name = $2;

-- name: ListPoliciesByGroupID :many
SELECT * FROM policies JOIN entity_policies ON policies.id = entity_policies.policy_id
WHERE policies.group_id = $1;

-- name: DeletePolicy :exec
DELETE FROM policies
WHERE id = $1;
