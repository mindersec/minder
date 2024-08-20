-- CreateEntity adds an entry to the entity_instances table so it can be tracked by Minder.

-- name: CreateEntity :one
INSERT INTO entity_instances (
    entity_type,
    name,
    project_id,
    provider_id,
    originated_from
) VALUES ($1, $2, sqlc.arg(project_id), sqlc.arg(provider_id), sqlc.narg(originated_from))
RETURNING *;

-- CreateEntityWithID adds an entry to the entities table with a specific ID so it can be tracked by Minder.

-- name: CreateEntityWithID :one
INSERT INTO entity_instances (
    id,
    entity_type,
    name,
    project_id,
    provider_id,
    originated_from
) VALUES ($1, $2, $3, sqlc.arg(project_id), sqlc.arg(provider_id), sqlc.narg(originated_from))
RETURNING *;


-- CreateOrEnsureEntityByID adds an entry to the entity_instances table if it does not exist, or returns the existing entry.

-- name: CreateOrEnsureEntityByID :one
INSERT INTO entity_instances (
    id,
    entity_type,
    name,
    project_id,
    provider_id,
    originated_from
) VALUES ($1, $2, $3, sqlc.arg(project_id), sqlc.arg(provider_id), sqlc.narg(originated_from))
ON CONFLICT (id) DO UPDATE
SET
    id = entity_instances.id  -- This is a "noop" update to ensure the RETURNING clause works
RETURNING *;

-- DeleteEntity removes an entity from the entity_instances table for a project.

-- name: DeleteEntity :exec
DELETE FROM entity_instances
WHERE id = $1 AND project_id = $2;

-- DeleteEntityByName removes an entity from the entity_instances table for a project.

-- name: DeleteEntityByName :exec
DELETE FROM entity_instances
WHERE name = sqlc.arg(name) AND project_id = $1;

-- GetEntityByID retrieves an entity by its ID for a project or hierarchy of projects.

-- name: GetEntityByID :one
SELECT * FROM entity_instances
WHERE entity_instances.id = $1 AND entity_instances.project_id = ANY(sqlc.arg(projects)::uuid[])
LIMIT 1;

-- GetEntityByName retrieves an entity by its name for a project or hierarchy of projects.
-- name: GetEntityByName :one
SELECT * FROM entity_instances
WHERE entity_instances.name = sqlc.arg(name) AND entity_instances.project_id = $1 AND entity_instances.entity_type = $2
LIMIT 1;

-- GetEntitiesByType retrieves all entities of a given type for a project or hierarchy of projects.
-- this is how one would get all repositories, artifacts, etc.

-- name: GetEntitiesByType :many
SELECT * FROM entity_instances
WHERE entity_instances.entity_type = $1 AND entity_instances.project_id = ANY(sqlc.arg(projects)::uuid[]);

-- name: TemporaryPopulateRepositories :exec
INSERT INTO entity_instances (id, entity_type, name, project_id, provider_id, created_at)
SELECT id, 'repository', repo_owner || '/' || repo_name, project_id, provider_id, created_at FROM repositories
WHERE NOT EXISTS (SELECT 1 FROM entity_instances WHERE entity_instances.id = repositories.id AND entity_instances.entity_type = 'repository');

-- name: TemporaryPopulateArtifacts :exec
INSERT INTO entity_instances (id, entity_type, name, project_id, provider_id, created_at, originated_from)
SELECT artifacts.id, 'artifact', LOWER(repositories.repo_owner) || '/' || artifacts.artifact_name, repositories.project_id, repositories.provider_id, artifacts.created_at, artifacts.repository_id FROM artifacts
JOIN repositories ON repositories.id = artifacts.repository_id
WHERE NOT EXISTS (SELECT 1 FROM entity_instances WHERE entity_instances.id = artifacts.id AND entity_instances.entity_type = 'artifact');

-- name: TemporaryPopulatePullRequests :exec
INSERT INTO entity_instances (id, entity_type, name, project_id, provider_id, created_at, originated_from)
SELECT pull_requests.id, 'pull_request', repositories.repo_owner || '/' || repositories.repo_name || '/' || pull_requests.pr_number::TEXT, repositories.project_id, repositories.provider_id, pull_requests.created_at, pull_requests.repository_id FROM pull_requests
JOIN repositories ON repositories.id = pull_requests.repository_id
WHERE NOT EXISTS (SELECT 1 FROM entity_instances WHERE entity_instances.id = pull_requests.id AND entity_instances.entity_type = 'pull_request');

-- name: GetProperty :one
SELECT * FROM properties
WHERE entity_id = $1 AND key = $2;

-- name: DeleteProperty :exec
DELETE FROM properties
WHERE entity_id = $1 AND key = $2;

-- name: UpsertProperty :one
INSERT INTO properties (
    entity_id,
    key,
    value,
    updated_at
) VALUES ($1, $2, $3, NOW())
ON CONFLICT (entity_id, key) DO UPDATE
    SET
        value = sqlc.arg(value),
        updated_at = NOW()
RETURNING *;

-- name: GetAllPropertiesForEntity :many
SELECT * FROM properties
WHERE entity_id = $1;

-- name: DeleteAllPropertiesForEntity :exec
DELETE FROM properties
WHERE entity_id = $1;