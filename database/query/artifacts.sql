-- name: CreateArtifact :one
INSERT INTO artifacts (
    repository_id,
    artifact_name,
    artifact_type,
    artifact_visibility) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpsertArtifact :one
INSERT INTO artifacts (
    repository_id,
    artifact_name,
    artifact_type,
    artifact_visibility
) VALUES ($1, $2, $3, $4)
ON CONFLICT (repository_id, LOWER(artifact_name))
DO UPDATE SET
    artifact_type = $3,
    artifact_visibility = $4
WHERE artifacts.repository_id = $1 AND artifacts.artifact_name = $2
RETURNING *;

-- name: GetArtifactByID :one
SELECT artifacts.id, artifacts.repository_id, artifacts.artifact_name, artifacts.artifact_type,
artifacts.artifact_visibility, artifacts.created_at,
repositories.provider, repositories.project_id, repositories.repo_owner, repositories.repo_name
FROM artifacts INNER JOIN repositories ON repositories.id = artifacts.repository_id
WHERE artifacts.id = $1;

-- name: GetArtifactByName :one
SELECT artifacts.id, artifacts.repository_id, artifacts.artifact_name, artifacts.artifact_type,
       artifacts.artifact_visibility, artifacts.created_at,
       repositories.provider, repositories.project_id, repositories.repo_owner, repositories.repo_name
FROM artifacts INNER JOIN repositories ON repositories.id = artifacts.repository_id
WHERE lower(artifacts.artifact_name) = lower(sqlc.arg(artifact_name)) AND artifacts.repository_id = $1;

-- name: ListArtifactsByRepoID :many
SELECT * FROM artifacts
WHERE repository_id = $1
ORDER BY id;

-- name: DeleteArtifact :exec
DELETE FROM artifacts
WHERE id = $1;