-- name: CreateArtifact :one
INSERT INTO artifacts (
    repository_id,
    artifact_name,
    artifact_type,
    artifact_visibility,
    project_id) VALUES ($1, $2, $3, $4, sqlc.arg(project_id)) RETURNING *;

-- name: UpsertArtifact :one
INSERT INTO artifacts (
    repository_id,
    artifact_name,
    artifact_type,
    artifact_visibility,
    project_id
) VALUES ($1, $2, $3, $4, sqlc.arg(project_id))
ON CONFLICT (project_id, LOWER(artifact_name))
DO UPDATE SET
    artifact_type = $3,
    artifact_visibility = $4
WHERE artifacts.repository_id = $1 AND artifacts.artifact_name = $2
RETURNING *;

-- name: GetArtifactByID :one
SELECT * FROM artifacts 
WHERE artifacts.id = $1 AND artifacts.project_id = $2;

-- name: GetArtifactByName :one
SELECT * FROM artifacts 
WHERE lower(artifacts.artifact_name) = lower(sqlc.arg(artifact_name)) AND artifacts.project_id = $1;

-- name: ListArtifactsByProjectID :many
SELECT * FROM artifacts
WHERE project_id = $1;

-- name: ListArtifactsByRepoID :many
SELECT * FROM artifacts
WHERE repository_id = $1
ORDER BY id;

-- name: DeleteArtifact :exec
DELETE FROM artifacts
WHERE id = $1;