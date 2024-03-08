-- name: CreateRuleType :one
INSERT INTO rule_type (
    name,
    provider,
    project_id,
    description,
    guidance,
    definition,
    severity_value,
    provider_id
    ) VALUES ($1, $2, $3, $4, $5, sqlc.arg(definition)::jsonb, sqlc.arg(severity_value), sqlc.arg(provider_id)) RETURNING *;

-- name: ListRuleTypesByProviderAndProject :many
SELECT * FROM rule_type WHERE provider = $1 AND project_id = $2;

-- name: GetRuleTypeByID :one
SELECT * FROM rule_type WHERE id = $1;

-- name: GetRuleTypeByName :one
SELECT * FROM rule_type WHERE provider = $1 AND project_id = $2 AND name = $3;

-- name: DeleteRuleType :exec
DELETE FROM rule_type WHERE id = $1;

-- name: UpdateRuleType :exec
UPDATE rule_type SET description = $2, definition = sqlc.arg(definition)::jsonb, severity_value = sqlc.arg(severity_value) WHERE id = $1;
