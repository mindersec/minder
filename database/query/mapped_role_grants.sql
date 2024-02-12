-- AddMappedRoleGrant adds a new mapped role grant to the database.

-- name: AddMappedRoleGrant :one
INSERT INTO mapped_role_grants (
    project_id,
    role,
    claim_mappings
) VALUES ($1, $2, $3) RETURNING *;

-- DeleteMappedRoleGrant deletes a mapped role grant from the database.

-- name: DeleteMappedRoleGrant :one
DELETE FROM mapped_role_grants
WHERE id = $1 AND project_id = $2 RETURNING *;

-- ResolveMappedRoleGrant resolves the subject of a mapped role grant.

-- name: ResolveMappedRoleGrant :one
UPDATE mapped_role_grants SET resolved_subject = $1 WHERE id = $2 RETURNING *;

-- SearchUnresolvedMappedRoleGrants searches for unresolved mapped role grants using
-- the provided claim mappings.

-- name: SearchUnresolvedMappedRoleGrants :many
SELECT * FROM mapped_role_grants
WHERE sqlc.arg(claim_mappings)::jsonb @> claim_mappings AND resolved_subject IS NULL;

-- ListMappedRoleGrants retrieves all mapped role grants from the database.

-- name: ListMappedRoleGrants :many
SELECT * FROM mapped_role_grants WHERE project_id = $1;

-- GetMappedRoleGrant retrieves a mapped role grant from the database.

-- name: GetMappedRoleGrant :one
SELECT * FROM mapped_role_grants
WHERE project_id = $1 AND role = $2 AND resolved_subject = $3;

-- ListResolvedMappedRoleGrantsForProject retrieves all resolved mapped role grants for a given project

-- name: ListResolvedMappedRoleGrantsForProject :many
SELECT * FROM mapped_role_grants
WHERE project_id = $1 AND resolved_subject IS NOT NULL;

-- ListUnresolvedMappedRoleGrantsForProject retrieves all unresolved mapped role grants for a given project

-- name: ListUnresolvedMappedRoleGrantsForProject :many
SELECT * FROM mapped_role_grants
WHERE project_id = $1 AND resolved_subject IS NULL;