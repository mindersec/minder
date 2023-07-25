-- name: UpdatePolicyStatus :exec
INSERT INTO policy_status (policy_id, policy_status, last_updated) VALUES ($1, $2, NOW())
ON CONFLICT (policy_id) DO UPDATE SET policy_status = $2, last_updated = NOW();

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