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
    default_branch,
    license,
    provider_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, sqlc.arg(default_branch), sqlc.arg(license), sqlc.arg(provider_id))
RETURNING *;

-- name: GetRepositoryByRepoID :one
SELECT * FROM repositories WHERE repo_id = $1;

-- name: GetRepositoryByRepoName :one
SELECT r.* FROM repositories AS r
JOIN providers AS p ON p.id = r.provider_id
WHERE r.repo_owner = $1 AND r.repo_name = $2 AND r.project_id = $3
  AND (lower(p.name) = lower(sqlc.narg('provider')::text) OR sqlc.narg('provider')::text IS NULL);

-- for backwards compatibility purposes only, we should not use this in new places
-- name: GetRepositoryAndProviderNameByRepoName :one
SELECT sqlc.embed(r), p.name AS provider_name FROM repositories AS r
JOIN providers AS p ON p.id = r.provider_id
WHERE r.repo_owner = $1 AND r.repo_name = $2 AND r.project_id = $3
  AND (lower(p.name) = lower(sqlc.narg('provider')::text) OR sqlc.narg('provider')::text IS NULL);

-- avoid using this, where possible use GetRepositoryByIDAndProject instead
-- name: GetRepositoryByID :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByIDAndProject :one
SELECT * FROM repositories WHERE id = $1 AND project_id = $2;

-- for backwards compatibility purposes only, we should not use this in new places
-- name: GetRepositoryAndProviderNameByIDAndProject :one
SELECT sqlc.embed(r), p.name AS provider_name FROM repositories as R
INNER JOIN providers AS p ON p.id = r.provider_id
WHERE r.id = $1 AND r.project_id = $2;

-- name: ListRepositoriesByProjectID :many
SELECT r.* FROM repositories as r
JOIN providers AS p ON p.id = r.provider_id
WHERE r.project_id = $1
AND (r.repo_id >= sqlc.narg('repo_id') OR sqlc.narg('repo_id') IS NULL)
AND lower(p.name) = lower(COALESCE(sqlc.narg('provider'), p.name)::text)
ORDER BY r.project_id, r.provider_id, r.repo_id
LIMIT sqlc.narg('limit')::bigint;

-- name: ListRegisteredRepositoriesByProjectIDAndProvider :many
SELECT sqlc.embed(r), p.name AS provider_name FROM repositories AS r
JOIN providers AS p ON p.id = r.provider_id
WHERE r.project_id = $1 AND r.webhook_id IS NOT NULL
  AND (lower(p.name) = lower(sqlc.narg('provider')::text) OR sqlc.narg('provider')::text IS NULL)
ORDER BY r.repo_name;

-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;

-- name: CountRepositories :one
SELECT COUNT(*) FROM repositories;