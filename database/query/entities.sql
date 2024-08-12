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

-- DeleteEntity removes an entity from the entity_instances table for a project.

-- name: DeleteEntity :exec
DELETE FROM entity_instances
WHERE id = $1 AND project_id = $2;

-- GetEntityByID retrieves an entity by its ID for a project or hierarchy of projects.

-- name: GetEntityByID :one
SELECT * FROM entity_instances
WHERE entity_instances.id = $1 AND entity_instances.project_id = ANY(sqlc.arg(projects)::uuid[])
LIMIT 1;

-- GetEntityByName retrieves an entity by its name for a project or hierarchy of projects.
SELECT * FROM entity_instances
WHERE lower(entity_instances.name) = lower(sqlc.arg(name)) AND entity_instances.project_id = $1
LIMIT 1;

-- GetEntitiesByType retrieves all entities of a given type for a project or hierarchy of projects.
-- this is how one would get all repositories, artifacts, etc.

-- name: GetEntitiesByType :many
SELECT * FROM entity_instances
WHERE entity_instances.entity_type = $1 AND entity_instances.project_id = ANY(sqlc.arg(projects)::uuid[]);

