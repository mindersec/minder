-- name: CreateProvider :one
INSERT INTO providers (
    name,
    project_id,
    implements,
    definition) VALUES ($1, $2, $3, sqlc.arg(definition)::jsonb) RETURNING *;

-- name: GetProviderByName :one
SELECT * FROM providers WHERE name = $1 AND project_id = $2;

-- name: GetProviderByID :one
SELECT * FROM providers WHERE id = $1 AND project_id = $2;

-- name: ListProvidersByProjectID :many
SELECT * FROM providers WHERE project_id = $1;

-- name: GlobalListProviders :many
SELECT * FROM providers;

-- name: DeleteProvider :exec
DELETE FROM providers WHERE id = $1 AND project_id = $2;