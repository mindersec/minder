-- CreateAcceptedRisk adds an accepted risk for a project
-- name: CreateAcceptedRisk :one
INSERT INTO accepted_risks (
    project_id,
    provider_id,
    entity_name,
    rule_type_id,
    expires_at
) VALUES (
    sqlc.arg(project_id),
    sqlc.arg(provider_id),
    sqlc.arg(entity_name),
    sqlc.arg(rule_type_id),
    sqlc.arg(expires_at)
)
RETURNING *;

-- ListAcceptedRisks lists active accepted risks for a project
-- name: ListAcceptedRisks :many
SELECT *
FROM accepted_risks
WHERE project_id = sqlc.arg(project_id)
  AND expires_at > NOW()
ORDER BY created_at DESC;

-- DeleteAcceptedRisk removes an accepted risk from a project
-- name: DeleteAcceptedRisk :exec
DELETE FROM accepted_risks
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);
