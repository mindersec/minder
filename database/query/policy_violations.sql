-- name: CreatePolicyViolation :one
INSERT INTO policy_violations (  
    repository_id,
    policy_id,
    metadata,
    violation) VALUES ($1, $2, sqlc.arg(metadata)::jsonb, sqlc.arg(violation)::jsonb) RETURNING *;

-- name: GetPolicyViolationsById :many
SELECT pt.policy_type, r.id as repo_id, r.repo_owner, r.repo_name,
pv.metadata, pv.violation, pv.created_at FROM policy_violations pv
INNER JOIN policies p ON p.id = pv.policy_id
INNER JOIN repositories r ON r.id = pv.repository_id
INNER JOIN policy_types pt ON pt.id = p.policy_type
WHERE p.id = $1 ORDER BY pv.created_at DESC LIMIT $2 OFFSET $3;

-- name: GetPolicyViolationsByGroup :many
SELECT pt.policy_type, r.id as repo_id, r.repo_owner, r.repo_name,
pv.metadata, pv.violation, pv.created_at FROM policy_violations pv
INNER JOIN policies p ON p.id = pv.policy_id
INNER JOIN repositories r ON r.id = pv.repository_id
INNER JOIN policy_types pt ON pt.id = p.policy_type
WHERE p.provider=$1 AND p.group_id=$2 ORDER BY pv.created_at DESC LIMIT $3 OFFSET $4;

-- name: GetPolicyViolationsByRepositoryId :many
SELECT pt.policy_type, r.id as repo_id, r.repo_owner, r.repo_name,
pv.metadata, pv.violation, pv.created_at FROM policy_violations pv
INNER JOIN policies p ON p.id = pv.policy_id
INNER JOIN repositories r ON r.id = pv.repository_id
INNER JOIN policy_types pt ON pt.id = p.policy_type
WHERE r.id = $1 ORDER BY pv.created_at DESC LIMIT $2 OFFSET $3;
