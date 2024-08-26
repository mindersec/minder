-- name: GetArtifactByID :one
SELECT * FROM artifacts
WHERE artifacts.id = $1 AND artifacts.project_id = $2;

-- name: GetArtifactByName :one
SELECT * FROM artifacts 
WHERE lower(artifacts.artifact_name) = lower(sqlc.arg(artifact_name))
AND artifacts.repository_id = $1 AND artifacts.project_id = $2;

-- name: ListArtifactsByRepoID :many
SELECT * FROM artifacts
WHERE repository_id = $1
ORDER BY id;