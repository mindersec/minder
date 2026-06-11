-- CreateEntity adds an entry to the entity_instances table so it can be tracked by Minder.
-- name: CreateEntity :one
INSERT INTO entity_instances (
    entity_type,
    name,
    project_id,
    provider_id,
    originated_from
) VALUES (
    sqlc.arg(entity_type), 
    sqlc.arg(name), 
    sqlc.arg(project_id), 
    sqlc.arg(provider_id), 
    sqlc.narg(originated_from)
)
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
) VALUES (
    sqlc.arg(id), 
    sqlc.arg(entity_type), 
    sqlc.arg(name), 
    sqlc.arg(project_id), 
    sqlc.arg(provider_id), 
    sqlc.narg(originated_from)
)
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
) VALUES (
    sqlc.arg(id), 
    sqlc.arg(entity_type), 
    sqlc.arg(name), 
    sqlc.arg(project_id), 
    sqlc.arg(provider_id), 
    sqlc.narg(originated_from)
)
ON CONFLICT (id) DO UPDATE
SET
    id = entity_instances.id  -- This is a "noop" update to ensure the RETURNING clause works
RETURNING *;

-- DeleteEntity removes an entity from the entity_instances table securely.
-- name: DeleteEntity :exec
DELETE FROM entity_instances
WHERE id = sqlc.arg(id) 
  AND project_id = sqlc.arg(project_id)
  AND provider_id = sqlc.arg(provider_id);

-- GetEntityByID retrieves an entity by its ID for a project or hierarchy of projects.
-- name: GetEntityByID :one
SELECT * FROM entity_instances
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id)
  AND provider_id = sqlc.arg(provider_id)
LIMIT 1;

-- GetEntityByName retrieves an entity by its name securely.
-- name: GetEntityByName :one
SELECT * FROM entity_instances
WHERE name = sqlc.arg(name)
  AND entity_type = sqlc.arg(entity_type)
  AND project_id = sqlc.arg(project_id)
  AND provider_id = sqlc.arg(provider_id)
LIMIT 1;

-- GetEntitiesByType retrieves all entities of a given type for a project hierarchy.
-- this is how one would get all repositories, artifacts, etc.
-- name: GetEntitiesByType :many
SELECT * FROM entity_instances
WHERE entity_type = sqlc.arg(entity_type)
  AND provider_id = sqlc.arg(provider_id)
  AND project_id = ANY(sqlc.arg(projects)::uuid[]);

-- ListEntitiesAfterID retrieves entities for pagination securely.
-- This is used for cursor-based iteration over all entities (e.g., in the reminder service).
-- name: ListEntitiesAfterID :many
SELECT * FROM entity_instances
WHERE entity_type = sqlc.arg(entity_type)
  AND id > sqlc.arg(id)
  AND provider_id = sqlc.arg(provider_id)
  AND project_id = ANY(sqlc.arg(projects)::uuid[])
ORDER BY id
LIMIT sqlc.arg('limit')::bigint;

-- EntityExistsAfterID checks if any entity exists after a cursor ID securely.
-- name: EntityExistsAfterID :one
SELECT EXISTS (
    SELECT 1
    FROM entity_instances
    WHERE entity_type = sqlc.arg(entity_type)
      AND id > sqlc.arg(id)
      AND provider_id = sqlc.arg(provider_id)
      AND project_id = ANY(sqlc.arg(projects)::uuid[])
) AS exists;

-- GetEntitiesByProvider retrieves all entities of a given provider scoped by project hierarchy.
-- this is how one would get all repositories, artifacts, etc. for a given provider.
-- name: GetEntitiesByProvider :many
SELECT * FROM entity_instances
WHERE provider_id = sqlc.arg(provider_id)
  AND project_id = ANY(sqlc.arg(projects)::uuid[]);

-- GetEntitiesByProjectHierarchy retrieves all entities for a project or hierarchy of projects.
-- name: GetEntitiesByProjectHierarchy :many
SELECT * FROM entity_instances
WHERE project_id = ANY(sqlc.arg(projects)::uuid[]);

-- CountEntitiesByType counts all entities of a given type (Global admin metric).
-- name: CountEntitiesByType :one
SELECT COUNT(*) FROM entity_instances
WHERE entity_type = sqlc.arg(entity_type);

-- CountEntitiesByTypeAndProject counts entities of a given type for a specific project.
-- name: CountEntitiesByTypeAndProject :one
SELECT COUNT(*) FROM entity_instances
WHERE entity_type = sqlc.arg(entity_type) 
  AND project_id = sqlc.arg(project_id);

-- GetProperty retrieves a single property, using a JOIN to ensure the caller owns the parent entity
-- name: GetProperty :one
SELECT p.* FROM properties p
JOIN entity_instances ei ON p.entity_id = ei.id
WHERE p.entity_id = sqlc.arg(entity_id) 
  AND p.key = sqlc.arg(key)
  AND ei.project_id = sqlc.arg(project_id)
  AND ei.provider_id = sqlc.arg(provider_id);

-- DeleteProperty deletes a property, using USING to ensure the caller owns the parent entity.
-- name: DeleteProperty :exec
DELETE FROM properties p
USING entity_instances ei
WHERE p.entity_id = ei.id 
  AND p.entity_id = sqlc.arg(entity_id) 
  AND p.key = sqlc.arg(key)
  AND ei.project_id = sqlc.arg(project_id)
  AND ei.provider_id = sqlc.arg(provider_id);

-- UpsertProperty upserts a property. 
-- NOTE: Ownership MUST be verified in Go (e.g. via GetEntityByID) before executing this statement.
-- name: UpsertProperty :one
INSERT INTO properties (
    entity_id,
    key,
    value,
    updated_at
) VALUES (
    sqlc.arg(entity_id), 
    sqlc.arg(key), 
    sqlc.arg(value), 
    NOW()
)
ON CONFLICT (entity_id, key) DO UPDATE
SET
    value = sqlc.arg(value),
    updated_at = NOW()
RETURNING *;

-- GetAllPropertiesForEntity retrieves all properties for one entity, strictly bounded.
-- name: GetAllPropertiesForEntity :many
SELECT p.* FROM properties p
JOIN entity_instances ei ON p.entity_id = ei.id
WHERE p.entity_id = sqlc.arg(entity_id)
  AND ei.project_id = sqlc.arg(project_id)
  AND ei.provider_id = sqlc.arg(provider_id);

-- GetPropertiesForEntities retrieves properties for multiple entities in bulk
-- name: GetPropertiesForEntities :many
SELECT p.* FROM properties p
JOIN entity_instances ei ON p.entity_id = ei.id
WHERE p.entity_id = ANY(sqlc.arg(entity_ids)::uuid[])
  AND ei.project_id = ANY(sqlc.arg(projects)::uuid[])
  AND ei.provider_id = sqlc.arg(provider_id);

-- DeleteAllPropertiesForEntity deletes all properties for an entity securely.
-- name: DeleteAllPropertiesForEntity :exec
DELETE FROM properties p
USING entity_instances ei
WHERE p.entity_id = ei.id 
  AND p.entity_id = sqlc.arg(entity_id)
  AND ei.project_id = sqlc.arg(project_id)
  AND ei.provider_id = sqlc.arg(provider_id);

-- GetTypedEntitiesByProperty retrieves entities matching a specific property JSONB query securely.
-- name: GetTypedEntitiesByProperty :many
SELECT ei.*
FROM entity_instances ei
JOIN properties p ON ei.id = p.entity_id
WHERE ei.entity_type = sqlc.arg(entity_type)
  AND (sqlc.arg(project_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR ei.project_id = sqlc.arg(project_id))
  AND (sqlc.arg(provider_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR ei.provider_id = sqlc.arg(provider_id))
  AND p.key = sqlc.arg(key)
  AND p.value @> sqlc.arg(value)::jsonb;