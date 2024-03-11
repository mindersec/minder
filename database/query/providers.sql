-- name: CreateProvider :one
INSERT INTO providers (
    name,
    project_id,
    implements,
    definition,
    auth_flows
    ) VALUES ($1, $2, $3, sqlc.arg(definition)::jsonb, sqlc.arg(auth_flows)) RETURNING *;

-- name: GetProviderByName :one
SELECT * FROM providers WHERE name = $1 AND project_id = $2;

-- name: GetProviderByID :one
SELECT * FROM providers WHERE id = $1 AND project_id = $2;

-- ListProvidersByProjectID allows us to lits all providers for a given project.

-- name: ListProvidersByProjectID :many
SELECT * FROM providers WHERE project_id = $1;

-- ListProvidersByProjectIDPaginated allows us to lits all providers for a given project
-- with pagination taken into account. In this case, the cursor is the creation date.

-- name: ListProvidersByProjectIDPaginated :many
SELECT * FROM providers
WHERE project_id = $1
    AND (created_at > sqlc.narg('created_at') OR sqlc.narg('created_at') IS NULL)
ORDER BY created_at DESC, id
LIMIT sqlc.arg('limit');


-- name: GlobalListProviders :many
SELECT * FROM providers;

-- name: GlobalListProvidersByName :many
SELECT * FROM providers WHERE name = $1;

-- name: UpdateProvider :exec
UPDATE providers
    SET implements = sqlc.arg(implements), definition = sqlc.arg(definition)::jsonb, auth_flows = sqlc.arg('auth_flows')
    WHERE id = sqlc.arg('id') AND project_id = sqlc.arg('project_id');

-- name: DeleteProvider :exec
DELETE FROM providers WHERE id = $1 AND project_id = $2;