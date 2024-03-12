-- name: CreateProvider :one
INSERT INTO providers (
    name,
    project_id,
    implements,
    definition,
    auth_flows
    ) VALUES ($1, $2, $3, sqlc.arg(definition)::jsonb, sqlc.arg(auth_flows)) RETURNING *;

-- GetProviderByName allows us to get a provider by its name. This takes
-- into account the project hierarchy, so it will only return the provider
-- if it exists in the project or any of its ancestors. It'll return the first
-- provider that matches the name.

-- name: GetProviderByName :one
SELECT * FROM providers WHERE name = $1 AND project_id = ANY(sqlc.arg(projects)::uuid[])
LIMIT 1;

-- name: GetProviderByID :one
SELECT * FROM providers WHERE id = $1;

-- ListProvidersByProjectID allows us to list all providers
-- for a given array of projects.

-- name: ListProvidersByProjectID :many
SELECT * FROM providers WHERE project_id = ANY(sqlc.arg(projects)::uuid[]);

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