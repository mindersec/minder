-- name: CreateRepository :one
INSERT INTO repositories (
    provider,
    group_id,
    repo_owner, 
    repo_name,
    repo_id,
    is_private,
    is_fork,
    webhook_id,
    webhook_url,
    deploy_url,
    clone_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING *;

-- name: GetRepositoryByID :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByRepoID :one
SELECT * FROM repositories WHERE provider = $1 AND repo_id = $2;

-- name: GetRepositoryByRepoName :one
SELECT * FROM repositories WHERE provider = $1 AND repo_owner = $2 AND repo_name = $3;

-- name: GetRepositoryByIDAndGroup :one
SELECT * FROM repositories WHERE provider = $1 AND repo_id = $2 AND group_id = $3;

-- name: ListRepositoriesByGroupID :many
SELECT * FROM repositories
WHERE provider = $1 AND group_id = $2
ORDER BY id
LIMIT $3
OFFSET $4;

-- name: ListRegisteredRepositoriesByGroupIDAndProvider :many
SELECT * FROM repositories
WHERE provider = $1 AND group_id = $2 AND webhook_id IS NOT NULL
ORDER BY id;

-- name: ListRepositoriesByOwner :many
SELECT * FROM repositories
WHERE provider = $1 AND repo_owner = $2
ORDER BY id
LIMIT $3
OFFSET $4;

-- name: ListAllRepositories :many
SELECT * FROM repositories WHERE provider = $1
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
provider = $11,
clone_url = $12,
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
provider = $10,
clone_url = $11,
updated_at = NOW() 
WHERE repo_id = $1 RETURNING *;


-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;