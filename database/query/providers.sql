-- name: CreateProvider :one
INSERT INTO providers (
    name,
    project_id,
    class,
    implements,
    definition,
    auth_flows
    ) VALUES ($1, $2, $3, $4, sqlc.arg(definition)::jsonb, sqlc.arg(auth_flows)) RETURNING *;

-- GetProviderByName allows us to get a provider by its name. This takes
-- into account the project hierarchy, so it will only return the provider
-- if it exists in the project or any of its ancestors. It'll return the first
-- provider that matches the name.

-- name: GetProviderByName :one
SELECT * FROM providers WHERE lower(name) = lower(sqlc.arg(name)) AND project_id = ANY(sqlc.arg(projects)::uuid[])
LIMIT 1;

-- name: GetProviderByID :one
SELECT * FROM providers WHERE id = $1;

-- FindProviders allows us to take a trait and filter
-- providers by it. It also optionally takes a name, in case we want to
-- filter by name as well.

-- name: FindProviders :many
SELECT * FROM providers
WHERE project_id = ANY(sqlc.arg(projects)::uuid[])
    AND (sqlc.narg('trait')::provider_type = ANY(implements) OR sqlc.narg('trait')::provider_type IS NULL)
    AND (lower(name) = lower(sqlc.narg('name')::text) OR sqlc.narg('name')::text IS NULL);

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
ORDER BY created_at ASC
LIMIT sqlc.arg('limit');

-- name: GlobalListProviders :many
SELECT * FROM providers;

-- name: GlobalListProvidersByClass :many
SELECT * FROM providers WHERE class = $1;

-- name: UpdateProvider :exec
UPDATE providers
    SET implements = sqlc.arg(implements), definition = sqlc.arg(definition)::jsonb, auth_flows = sqlc.arg('auth_flows')
    WHERE id = sqlc.arg('id') AND project_id = sqlc.arg('project_id');

-- name: DeleteProvider :exec
DELETE FROM providers WHERE id = $1;