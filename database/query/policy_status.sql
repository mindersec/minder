-- name: UpdatePolicyStatus :exec
INSERT INTO policy_status (repository_id, policy_id, policy_status, last_updated) VALUES ($1, $2, $3, NOW())
ON CONFLICT (repository_id, policy_id) DO UPDATE SET policy_status = $3, last_updated = NOW();

-- name: GetPolicyStatus :many
SELECT pt.policy_type, r.id as repo_id, r.repo_owner, r.repo_name,
ps.policy_status, ps.last_updated FROM policy_status ps
INNER JOIN policies p ON p.id = ps.policy_id
INNER JOIN repositories r ON r.id = ps.repository_id
INNER JOIN policy_types pt ON pt.id = p.policy_type
WHERE p.id = $1;