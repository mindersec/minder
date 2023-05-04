-- name: CreateOrganisation :one
INSERT INTO organisations (
    name,
    company
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetOrganisation :one 
SELECT * FROM organisations 
WHERE id = $1 LIMIT 1;

-- name: GetOrganisationForUpdate :one
SELECT * FROM organisations
WHERE id = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListOrganisations :many
SELECT * FROM organisations
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateOrganisation :one
UPDATE organisations
SET name = $2, company = $3, updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: DeleteOrganisation :exec
DELETE FROM organisations
WHERE id = $1;