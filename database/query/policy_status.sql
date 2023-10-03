-- name: UpdateRuleEvaluationStatusForRepository :exec
UPDATE rule_evaluation_status 
    SET eval_status = $1, eval_details = $2, remediation_status = $3, remediation_details=$4, remediation_last_updated=$5, last_updated = NOW()
    WHERE id = $5;

-- name: CreateRuleEvaluationStatusForRepository :exec
INSERT INTO rule_evaluation_status (
    policy_id,
    repository_id,
    rule_type_id,
    entity,
    eval_status,
    eval_details,
    last_updated
) VALUES ($1, $2, $3, 'repository', $4, $5, NOW());

-- name: UpsertRuleEvaluationStatus :exec
INSERT INTO rule_evaluation_status (
    policy_id,
    repository_id,
    artifact_id,
    rule_type_id,
    entity,
    eval_status,
    eval_details,
    remediation_status,
    remediation_details,
    remediation_last_updated,
    eval_last_updated
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
ON CONFLICT(policy_id, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), entity, rule_type_id) DO UPDATE SET
    eval_status = $6,
    eval_details = $7,
    remediation_status = $8,
    remediation_details = $9,
    remediation_last_updated = COALESCE($10, rule_evaluation_status.remediation_last_updated), -- don't overwrite timestamp already set with NULL
    eval_last_updated = NOW()
WHERE rule_evaluation_status.policy_id = $1
  AND rule_evaluation_status.repository_id = $2
  AND rule_evaluation_status.artifact_id IS NOT DISTINCT FROM $3
  AND rule_evaluation_status.rule_type_id = $4
  AND rule_evaluation_status.entity = $5;

-- name: GetPolicyStatusByIdAndProject :one
SELECT p.id, p.name, ps.policy_status, ps.last_updated FROM policy_status ps
INNER JOIN policies p ON p.id = ps.policy_id
WHERE p.id = $1 AND p.project_id = $2;

-- name: GetPolicyStatusByProject :many
SELECT p.id, p.name, ps.policy_status, ps.last_updated FROM policy_status ps
INNER JOIN policies p ON p.id = ps.policy_id
WHERE p.project_id = $1;

-- name: ListRuleEvaluationStatusByPolicyId :many
SELECT res.eval_status as eval_status, res.eval_last_updated as eval_last_updated, res.eval_details as eval_details, res.remediation_status as rem_status, res.remediation_details as rem_details, res.remediation_last_updated as rem_last_updated, res.repository_id as repository_id, res.entity as entity, repo.repo_name as repo_name, repo.repo_owner as repo_owner, repo.provider as provider, rt.name as rule_type_name, rt.id as rule_type_id
FROM rule_evaluation_status res
INNER JOIN repositories repo ON repo.id = res.repository_id
INNER JOIN rule_type rt ON rt.id = res.rule_type_id
WHERE res.policy_id = $1 AND
    (
        CASE
            WHEN sqlc.narg(entity_type)::entities = 'repository' AND res.repository_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_type)::entities  = 'artifact' AND res.artifact_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_id)::UUID IS NULL THEN true
            ELSE false
        END
    ) AND (rt.name = sqlc.narg(rule_name) OR sqlc.narg(rule_name) IS NULL)
;
