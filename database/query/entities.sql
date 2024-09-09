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
WHERE entity_instances.id = $1
LIMIT 1;

-- GetEntityByName retrieves an entity by its name for a project or hierarchy of projects.
-- name: GetEntityByName :one
SELECT * FROM entity_instances
WHERE
    entity_instances.name = sqlc.arg(name)
    AND entity_instances.project_id = $1
    AND entity_instances.entity_type = $2
    AND entity_instances.provider_id = sqlc.arg(provider_id)
LIMIT 1;

-- GetEntitiesByType retrieves all entities of a given type for a project or hierarchy of projects.
-- this is how one would get all repositories, artifacts, etc.

-- name: GetEntitiesByType :many
SELECT * FROM entity_instances
WHERE entity_instances.entity_type = $1 AND entity_instances.project_id = ANY(sqlc.arg(projects)::uuid[]);

-- GetEntitiesByProvider retrieves all entities of a given provider.
-- this is how one would get all repositories, artifacts, etc. for a given provider.

-- name: GetEntitiesByProvider :many
SELECT * FROM entity_instances
WHERE entity_instances.provider_id = $1;

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

-- name: GetTypedEntitiesByProperty :many
SELECT ei.*
FROM entity_instances ei
         JOIN properties p ON ei.id = p.entity_id
WHERE ei.entity_type = sqlc.arg(entity_type)
  AND (sqlc.arg(project_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR ei.project_id = sqlc.arg(project_id))
  AND p.key = sqlc.arg(key)
  AND p.value @> sqlc.arg(value)::jsonb;