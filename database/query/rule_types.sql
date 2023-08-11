-- name: CreateRuleType :one
INSERT INTO rule_type (
    name,
    provider,
    group_id,
    definition,
    params) VALUES ($1, $2, $3, sqlc.arg(definition)::jsonb, sqlc.arg(params)::jsonb) RETURNING *;

-- name: ListRuleTypesByProviderAndGroup :many
SELECT * FROM rule_type WHERE provider = $1 AND group_id = $2;

-- name: GetRuleTypeByID :one
SELECT * FROM rule_type WHERE id = $1;

-- name: GetRuleTypeByName :one
SELECT * FROM rule_type WHERE provider = $1 AND group_id = $2 AND name = $3;

-- name: DeleteRuleType :exec
DELETE FROM rule_type WHERE id = $1;

-- name: UpdateRuleType :exec
UPDATE rule_type SET definition = sqlc.arg(definition)::jsonb, params = sqlc.arg(params)::jsonb WHERE id = $1;
