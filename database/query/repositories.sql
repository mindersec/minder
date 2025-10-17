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

-- name: UpdateReminderLastSentForRepositories :exec
UPDATE repositories
SET reminder_last_sent = NOW()
WHERE id = ANY (sqlc.arg('repository_ids')::uuid[]);

-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;
