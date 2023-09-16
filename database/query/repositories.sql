-- name: CreateRepository :one
INSERT INTO repositories (
    provider,
    repo_owner, 
    repo_name,
    repo_id,
    is_private,
    is_fork,
    webhook_id,
    webhook_url,
    deploy_url,
    clone_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING *;

-- name: GetRepositoryByID :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByRepoID :one
SELECT * FROM repositories WHERE repo_id = $1;

-- name: GetRepositoryByRepoName :one
SELECT * FROM repositories WHERE provider = $1 AND repo_owner = $2 AND repo_name = $3;

-- name: GetRepositoryByIDAndProvider :one
SELECT * FROM repositories WHERE provider = $1 AND repo_id = $2;

-- name: ListRepositoriesByProvider :many
SELECT * FROM repositories
WHERE provider = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListRegisteredRepositoriesByProvider :many
SELECT * FROM repositories
WHERE provider = $1 AND webhook_id IS NOT NULL
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
SET repo_owner = $2,
repo_name = $3,
repo_id = $4,
is_private = $5,
is_fork = $6,
webhook_id = $7,
webhook_url = $8,
deploy_url = $9, 
provider = $10,
-- set clone_url if the value is not an empty string
clone_url = CASE WHEN sqlc.arg(clone_url)::text = '' THEN clone_url ELSE sqlc.arg(clone_url)::text END,
updated_at = NOW() 
WHERE id = $1 RETURNING *;

-- name: UpdateRepositoryByID :one
UPDATE repositories 
SET repo_owner = $2,
repo_name = $3,
is_private = $4,
is_fork = $5,
webhook_id = $6,
webhook_url = $7,
deploy_url = $8, 
provider = $9,
clone_url = CASE WHEN sqlc.arg(clone_url)::text = '' THEN clone_url ELSE sqlc.arg(clone_url)::text END,
updated_at = NOW() 
WHERE repo_id = $1 RETURNING *;


-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;