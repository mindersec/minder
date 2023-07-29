-- name: UpdateRuleEvaluationStatusForRepository :exec
UPDATE rule_evaluation_status 
    SET eval_status = $1, details = $2, last_updated = NOW()
    WHERE id = $3;

-- name: CreateRuleEvaluationStatusForRepository :exec
INSERT INTO rule_evaluation_status (
    policy_id,
    repository_id,
    rule_type_id,
    entity,
    eval_status,
    details,
    last_updated
) VALUES ($1, $2, $3, 'repository', $4, $5, NOW());

-- name: UpsertRuleEvaluationStatusForRepository :exec
INSERT INTO rule_evaluation_status (
    policy_id,
    repository_id,
    rule_type_id,
    entity,
    eval_status,
    details,
    last_updated
) VALUES ($1, $2, $3, 'repository', $4, $5, NOW())
ON CONFLICT(policy_id, repository_id, entity, rule_type_id) DO UPDATE SET
    eval_status = $4,
    details = $5,
    last_updated = NOW()
WHERE rule_evaluation_status.policy_id = $1
  AND rule_evaluation_status.repository_id = $2
  AND rule_evaluation_status.entity = 'repository'
  AND rule_evaluation_status.rule_type_id = $3;

-- name: GetRuleEvaluationStatusForRepository :one
SELECT * FROM rule_evaluation_status
WHERE policy_id = $1 AND entity = 'repository' AND repository_id = $2 AND rule_type_id = $3;

-- name: GetPolicyStatusByIdAndGroup :one
SELECT p.id, p.name, ps.policy_status, ps.last_updated FROM policy_status ps
INNER JOIN policies p ON p.id = ps.policy_id
WHERE p.id = $1 AND p.group_id = $2;

-- name: GetPolicyStatusByGroup :many
SELECT p.id, p.name, ps.policy_status, ps.last_updated FROM policy_status ps
INNER JOIN policies p ON p.id = ps.policy_id
WHERE p.group_id = $1;

-- name: ListRuleEvaluationStatusForRepositoriesByPolicyId :many
SELECT res.eval_status, res.last_updated, res.repository_id, repo.repo_name, repo.repo_owner, repo.provider, rt.name, rt.id as rule_type_id
FROM rule_evaluation_status res
INNER JOIN repositories repo ON repo.id = res.repository_id
INNER JOIN rule_type rt ON rt.id = res.rule_type_id
WHERE res.entity = 'repository' AND res.policy_id = $1;

-- name: ListRuleEvaluationStatusForRepositoryByPolicyId :many
SELECT res.eval_status, res.last_updated, res.repository_id, repo.repo_name, repo.repo_owner, repo.provider, rt.name, rt.id as rule_type_id
FROM rule_evaluation_status res
INNER JOIN repositories repo ON repo.id = res.repository_id
INNER JOIN rule_type rt ON rt.id = res.rule_type_id
WHERE res.entity = 'repository' AND res.policy_id = $1 AND repo.id = $2 ;