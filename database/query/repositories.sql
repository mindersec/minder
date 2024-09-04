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
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, sqlc.arg(default_branch), sqlc.arg(license), sqlc.arg(provider_id)) RETURNING *;

-- name: GetRepositoryByRepoID :one
SELECT * FROM repositories WHERE repo_id = $1;

-- name: GetRepositoryByRepoName :one
SELECT * FROM repositories
    WHERE repo_owner = $1 AND repo_name = $2 AND project_id = $3
    AND (lower(provider) = lower(sqlc.narg('provider')::text) OR sqlc.narg('provider')::text IS NULL);

-- avoid using this, where possible use GetRepositoryByIDAndProject instead
-- name: GetRepositoryByID :one
SELECT * FROM repositories WHERE id = $1;

-- name: GetRepositoryByIDAndProject :one
SELECT * FROM repositories WHERE id = $1 AND project_id = $2;

-- name: ListRepositoriesByProjectID :many
SELECT * FROM repositories
WHERE project_id = $1
  AND (repo_id >= sqlc.narg('repo_id') OR sqlc.narg('repo_id') IS NULL)
  AND lower(provider) = lower(COALESCE(sqlc.narg('provider'), provider)::text)
ORDER BY project_id, provider, repo_id
LIMIT sqlc.narg('limit')::bigint;

-- name: ListRegisteredRepositoriesByProjectIDAndProvider :many
SELECT * FROM repositories
WHERE project_id = $1 AND webhook_id IS NOT NULL
    AND (lower(provider) = lower(sqlc.narg('provider')::text) OR sqlc.narg('provider')::text IS NULL)
ORDER BY repo_name;

-- name: ListRepositoriesAfterID :many
SELECT *
FROM repositories
WHERE id > $1
ORDER BY id
LIMIT sqlc.arg('limit')::bigint;

-- name: UpdateReminderLastSentForRepositories :exec
UPDATE repositories
SET reminder_last_sent = NOW()
WHERE id = ANY (sqlc.arg('repository_ids')::uuid[]);

-- name: RepositoryExistsAfterID :one
SELECT EXISTS (
  SELECT 1
  FROM repositories
  WHERE id > $1)
AS exists;

-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;

-- name: CountRepositories :one
SELECT COUNT(*) FROM repositories;

-- name: CountRepositoriesByProjectID :one
SELECT COUNT(*)
FROM repositories
WHERE project_id = $1;

-- get a list of repos with webhooks belonging to a provider
-- is used for webhook cleanup during provider deletion
-- name: GetProviderWebhooks :many
SELECT repo_owner, repo_name, webhook_id FROM repositories
WHERE webhook_id IS NOT NULL AND provider_id = $1;

