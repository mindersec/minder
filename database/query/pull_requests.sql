-- name: GetPullRequest :one
SELECT * FROM pull_requests
WHERE repository_id = $1 AND pr_number = $2;

-- name: GetPullRequestByID :one
SELECT * FROM pull_requests
WHERE id = $1;