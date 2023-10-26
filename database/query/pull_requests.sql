-- name: CreatePullRequest :one
INSERT INTO pull_requests (
    repository_id,
    pr_number
) VALUES ($1, $2) RETURNING *;

-- name: UpsertPullRequest :one
INSERT INTO pull_requests (
    repository_id,
    pr_number
) VALUES ($1, $2)
ON CONFLICT (repository_id, pr_number)
DO UPDATE SET
    updated_at = NOW()
WHERE pull_requests.repository_id = $1 AND pull_requests.pr_number = $2
RETURNING *;

-- name: GetPullRequest :one
SELECT * FROM pull_requests
WHERE repository_id = $1 AND pr_number = $2;

-- name: GetPullRequestByID :one
SELECT * FROM pull_requests
WHERE id = $1;

-- name: DeletePullRequest :exec
DELETE FROM pull_requests
WHERE repository_id = $1 AND pr_number = $2;