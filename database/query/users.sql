-- name: CreateUser :one
INSERT INTO users (role_id, email, username, password, first_name, last_name, is_protected) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserClaims :one
SELECT u.id as user_id, u.role_id as role_id, r.is_admin as is_admin, r.group_id as group_id,
g.organization_id as organization_id, u.min_token_issued_time AS min_token_issued_time FROM users u
INNER JOIN roles r ON u.role_id = r.id INNER JOIN groups g ON r.group_id = g.id WHERE u.id = $1;

-- name: GetUserByUserName :one
SELECT * FROM users WHERE username = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE role_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListUsersByRoleID :many
SELECT * FROM users WHERE role_id = $1;

-- name: UpdateUser :one
UPDATE users SET role_id = $2, email = $3, username = $4, password = $5, first_name = $6, last_name = $7, is_protected = $8, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: RevokeUserToken :one
UPDATE users SET min_token_issued_time = NOW() WHERE id = $1 RETURNING *;

-- name: RevokeUsersTokens :one
UPDATE users SET min_token_issued_time = NOW() RETURNING *;

-- name: RevokeOrganizationUsersTokens :one
UPDATE users SET min_token_issued_time = NOW() WHERE role_id IN (SELECT id FROM roles WHERE group_id IN (SELECT id FROM groups WHERE organization_id = $1)) RETURNING *;

-- name: RevokeGroupUsersTokens :one
UPDATE users SET min_token_issued_time = NOW() WHERE role_id IN (SELECT id FROM roles WHERE group_id = $1) RETURNING *;

-- name: RevokeRoleUsersTokens :one
UPDATE users SET min_token_issued_time = NOW() WHERE role_id = $1 RETURNING *;