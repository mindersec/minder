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
SELECT * FROM artifacts WHERE id = $1;

-- name: GetArtifactByName :one
SELECT * FROM artifacts WHERE repository_id = $1 AND artifact_name = $2;

-- name: ListArtifactsByRepoID :many
SELECT * FROM artifacts
WHERE repository_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: DeleteArtifact :exec
DELETE FROM artifacts
WHERE id = $1;