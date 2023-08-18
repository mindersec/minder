-- name: CreateArtifactVersion :one
INSERT INTO artifact_versions (
    artifact_id,
    version,
    tags,
    sha,
    signature_verification,
    github_workflow, created_at) VALUES ($1, $2, $3, $4,
    sqlc.arg(signature_verification)::jsonb,
    sqlc.arg(github_workflow)::jsonb,
    $5) RETURNING *;

-- name: UpsertArtifactVersion :one
INSERT INTO artifact_versions (
    artifact_id,
    version,
    tags,
    sha,
    signature_verification,
    github_workflow,
    created_at
) VALUES ($1, $2, $3, $4,
    sqlc.arg(signature_verification)::jsonb,
    sqlc.arg(github_workflow)::jsonb,
    $5)
ON CONFLICT (artifact_id, sha)
DO UPDATE SET
    version = $2,
    tags = $3,
    signature_verification = sqlc.arg(signature_verification)::jsonb,
    github_workflow = sqlc.arg(github_workflow)::jsonb,
    created_at = $5
WHERE artifact_versions.artifact_id = $1 AND artifact_versions.sha = $4
RETURNING *;


-- name: GetArtifactVersionByID :one
SELECT * FROM artifact_versions WHERE id = $1;

-- name: GetArtifactVersionBySha :one
SELECT * FROM artifact_versions WHERE artifact_id = $1 AND sha = $2;

-- name: ListArtifactVersionsByArtifactID :many
SELECT * FROM artifact_versions
WHERE artifact_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListArtifactVersionsByArtifactIDAndTag :many
SELECT * FROM artifact_versions
WHERE artifact_id = $1
AND $2=ANY(STRING_TO_ARRAY(tags, ','))
ORDER BY created_at DESC
LIMIT $3;

-- name: DeleteArtifactVersion :exec
DELETE FROM artifact_versions
WHERE id = $1;

-- name: DeleteOldArtifactVersions :exec
DELETE FROM artifact_versions
WHERE artifact_id = $1 AND created_at <= $2;