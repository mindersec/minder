-- CreateException adds an exception for a project
-- name: CreateException :one
INSERT INTO exceptions (
    project_id,
    entity_name,
    entity_id,
    rule_type_id,
    expires_at
) VALUES (
    sqlc.arg(project_id),
    sqlc.arg(entity_name),
    sqlc.arg(entity_id),
    sqlc.arg(rule_type_id),
    sqlc.arg(expires_at)
)
ON CONFLICT (entity_id, rule_type_id)
DO UPDATE SET
    expires_at = EXCLUDED.expires_at
RETURNING *;

-- ListExceptions lists active exceptions for a project
-- name: ListExceptions :many
SELECT *
FROM exceptions
WHERE project_id = sqlc.arg(project_id)
  AND expires_at > NOW()
ORDER BY created_at DESC;

-- DeleteException removes an exception from a project
-- name: DeleteException :exec
DELETE FROM exceptions
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);
