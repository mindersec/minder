-- CreateException adds an exception for a project
-- name: CreateException :one
INSERT INTO exceptions (
    project_id,
    entity_id,
    profile_name,
    rule_name,
    reason,
    expires_at
) VALUES (
    sqlc.arg(project_id),
    sqlc.arg(entity_id),
    sqlc.arg(profile_name),
    sqlc.arg(rule_name),
    sqlc.arg(reason),
    sqlc.arg(expires_at)
)
ON CONFLICT (entity_id, lower(profile_name), lower(rule_name))
DO UPDATE SET
    expires_at = EXCLUDED.expires_at,
    reason = CASE
      WHEN sqlc.arg(reason_provided)::boolean
      THEN EXCLUDED.reason
      ELSE exceptions.reason
    END
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
