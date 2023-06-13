-- name: CreateRepository :one
INSERT INTO repositories (
    group_id,
    repo_owner, 
    repo_name,
    webhook_id,
    webhook_url,
    deploy_url) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetRepositoryByID :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByRepoName :one
SELECT * FROM repositories WHERE repo_name = $1;

-- name: ListRepositoriesByGroupID :many
SELECT * FROM repositories
WHERE group_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListRepositoriesByOwner :many
SELECT * FROM repositories
WHERE repo_owner = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateRepository :one
UPDATE repositories 
SET group_id = $2,
repo_owner = $3,
repo_name = $4,
webhook_id = $5,
webhook_url = $6,
deploy_url = $7, 
updated_at = NOW() 
WHERE id = $1 RETURNING *;

-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;
