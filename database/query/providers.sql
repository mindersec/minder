-- name: CreateProvider :one
INSERT INTO providers (
    name,
    group_id,
    implements,
    definition) VALUES ($1, $2, $3, sqlc.arg(definition)::jsonb) RETURNING *;

-- name: GetProviderByName :one
SELECT * FROM providers WHERE name = $1 AND group_id = $2;

-- name: GetProviderByID :one
SELECT * FROM providers WHERE id = $1 AND group_id = $2;

-- name: ListProvidersByGroupID :many
SELECT * FROM providers WHERE group_id = $1;

-- name: GlobalListProviders :many
SELECT * FROM providers;

-- name: DeleteProvider :exec
DELETE FROM providers WHERE id = $1 AND group_id = $2;