-- name: CreateProject :one
INSERT INTO projects (
    name,
    parent_id,
    metadata
) VALUES (
    $1, $2, sqlc.arg(metadata)::jsonb
) RETURNING *;

-- name: GetProjectByID :one
SELECT id, name, parent_id, metadata, created_at, updated_at FROM projects
WHERE id = $1;

-- name: GetParents :many
WITH RECURSIVE get_parents AS (
    SELECT id, parent_id, created_at FROM projects 
    WHERE projects.id = $1

    UNION

    (
        SELECT p.id, p.parent_id, p.created_at FROM projects p
        INNER JOIN get_parents gp ON p.id = gp.parent_id
        ORDER BY created_at ASC
    )
)
SELECT id FROM get_parents;

-- name: GetParentsUntil :many
WITH RECURSIVE get_parents_until AS (
    SELECT id, parent_id, created_at FROM projects 
    WHERE projects.id = $1

    UNION

    (
        SELECT p.id, p.parent_id, p.created_at FROM projects p
        INNER JOIN get_parents_until gpu ON p.id = gpu.parent_id
        WHERE p.id != $2
        ORDER BY created_at ASC
    )
)
SELECT id FROM get_parents_until;

-- name: GetChildren :many
WITH RECURSIVE get_children AS (
    SELECT id, parent_id, created_at FROM projects 
    WHERE projects.id = $1

    UNION

    (
        SELECT p.id, p.parent_id, p.created_at FROM projects p
        INNER JOIN get_children gc ON p.parent_id = gc.id
        ORDER BY created_at ASC
    )
)
SELECT id FROM get_children;

-- name: DeleteProject :many
WITH RECURSIVE get_children AS (
    SELECT id, parent_id FROM projects
    WHERE projects.id = $1 AND projects.parent_id IS NOT NULL

    UNION

    SELECT p.id, d.parent_id FROM projects d
    INNER JOIN get_children gc ON p.parent_id = gc.id
)
DELETE FROM projects
WHERE id IN (SELECT id FROM get_children)
RETURNING id, name, metadata, created_at, updated_at, parent_id;