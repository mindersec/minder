-- name: CreateOrganization :one
INSERT INTO projects (
    name,
    is_organization,
    metadata 
) VALUES (
    $1, TRUE, sqlc.arg(metadata)::jsonb
) RETURNING *;

-- name: GetOrganization :one 
SELECT * FROM projects 
WHERE id = $1 AND is_organization = TRUE LIMIT 1;

-- name: GetOrganizationByName :one 
SELECT * FROM projects
WHERE name = $1 AND is_organization = TRUE LIMIT 1;


-- name: GetOrganizationForUpdate :one
SELECT * FROM projects
WHERE name = $1 AND is_organization = TRUE LIMIT 1
FOR NO KEY UPDATE;

-- name: ListOrganizations :many
SELECT * FROM projects
WHERE is_organization = TRUE
ORDER BY name
LIMIT $1
OFFSET $2;

-- name: UpdateOrganization :one
UPDATE projects
SET name = $2, metadata = sqlc.arg(metadata), updated_at = NOW()
WHERE id = $1 AND is_organization = TRUE RETURNING *;

-- name: DeleteOrganization :exec
DELETE FROM projects
WHERE id = $1 AND is_organization = TRUE;