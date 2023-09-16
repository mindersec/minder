-- name: CreatePolicy :one
INSERT INTO policies (  
    provider,
    name) VALUES ($1, $2) RETURNING *;

-- name: CreatePolicyForEntity :one
INSERT INTO entity_policies (
    entity,
    policy_id,
    contextual_rules) VALUES ($1, $2, sqlc.arg(contextual_rules)::jsonb) RETURNING *;

-- name: GetPolicyByProviderAndID :many
SELECT * FROM policies JOIN entity_policies ON policies.id = entity_policies.policy_id
WHERE policies.provider = $1 AND policies.id = $2;

-- name: GetPolicyByID :one
SELECT * FROM policies WHERE id = $1;

-- name: GetPolicyByProviderAndName :many
SELECT * FROM policies JOIN entity_policies ON policies.id = entity_policies.policy_id
WHERE policies.provider = $1 AND policies.name = $2;

-- name: ListPoliciesByProvider :many
SELECT * FROM policies JOIN entity_policies ON policies.id = entity_policies.policy_id
WHERE policies.provider = $1;

-- name: DeletePolicy :exec
DELETE FROM policies
WHERE id = $1;
