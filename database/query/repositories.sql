-- name: CreateRepository :one
INSERT INTO repositories (
    provider,
    project_id,
    repo_owner, 
    repo_name,
    repo_id,
    is_private,
    is_fork,
    webhook_id,
    webhook_url,
    deploy_url,
    clone_url,
    default_branch) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, sqlc.arg(default_branch)) RETURNING *;

-- name: GetRepositoryByID :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByRepoID :one
SELECT * FROM repositories WHERE repo_id = $1;

-- name: GetRepositoryByRepoName :one
SELECT * FROM repositories WHERE provider = $1 AND repo_owner = $2 AND repo_name = $3 AND project_id = $4;

-- name: GetRepositoryByIDAndProject :one
SELECT * FROM repositories WHERE provider = $1 AND repo_id = $2 AND project_id = $3;

-- name: ListRepositoriesByProjectID :many
SELECT * FROM repositories
WHERE provider = $1 AND project_id = $2
  AND (repo_id >= sqlc.narg('repo_id') OR sqlc.narg('repo_id') IS NULL)
ORDER BY project_id, provider, repo_id
LIMIT sqlc.narg('limit');

-- name: ListRegisteredRepositoriesByProjectIDAndProvider :many
SELECT * FROM repositories
WHERE provider = $1 AND project_id = $2 AND webhook_id IS NOT NULL
ORDER BY repo_name;

-- name: UpdateRepository :one
UPDATE repositories 
SET project_id = $2,
repo_owner = $3,
repo_name = $4,
repo_id = $5,
is_private = $6,
is_fork = $7,
webhook_id = $8,
webhook_url = $9,
deploy_url = $10, 
provider = $11,
-- set clone_url if the value is not an empty string
clone_url = CASE WHEN sqlc.arg(clone_url)::text = '' THEN clone_url ELSE sqlc.arg(clone_url)::text END,
updated_at = NOW(),
default_branch = sqlc.arg(default_branch)
WHERE id = $1 RETURNING *;


-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;

-- name: CountRepositories :one
SELECT COUNT(*) FROM repositories;
