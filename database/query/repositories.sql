-- name: CreateRepository :one
INSERT INTO repositories (
    group_id,
    repo_owner, 
    repo_name,
    repo_id,
    is_private,
    is_fork,
    webhook_id,
    webhook_url,
    deploy_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: GetRepositoryByID :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByRepoID :one
SELECT * FROM repositories WHERE repo_id = $1;

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

-- name: ListAllRepositories :many
SELECT * FROM repositories
ORDER BY id;


-- name: UpdateRepository :one
UPDATE repositories 
SET group_id = $2,
repo_owner = $3,
repo_name = $4,
repo_id = $5,
is_private = $6,
is_fork = $7,
webhook_id = $8,
webhook_url = $9,
deploy_url = $10, 
updated_at = NOW() 
WHERE id = $1 RETURNING *;

-- name: UpdateRepositoryByID :one
UPDATE repositories 
SET group_id = $2,
repo_owner = $3,
repo_name = $4,
is_private = $5,
is_fork = $6,
webhook_id = $7,
webhook_url = $8,
deploy_url = $9, 
updated_at = NOW() 
WHERE repo_id = $1 RETURNING *;


-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;