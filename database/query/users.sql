-- name: CreateUser :one
INSERT INTO users (organization_id, email, username, password, first_name, last_name, is_protected, needs_password_change) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserClaims :one
SELECT u.id as user_id, g.organization_id as organization_id, u.min_token_issued_time AS min_token_issued_time,
u.needs_password_change as needs_password_change,
ARRAY_AGG(jsonb_build_object('role_id', roles.id, 'is_admin', roles.is_admin, 'group_id', roles.group_id, 'organization_id', roles.organization_id)) AS role_info,
ARRAY_AGG(g.id) AS group_ids
 FROM users u
INNER JOIN roles r ON u.role_id = r.id INNER JOIN groups g ON r.group_id = g.id WHERE u.id = $1;

-- name: GetUserByUserName :one
SELECT * FROM users WHERE username = $1;

-- name: UpdateUser :one
UPDATE users SET email = $2, username = $3, password = $4, first_name = $5, last_name = $6, is_protected = $7, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: UpdatePassword :one
UPDATE users SET password = $2, needs_password_change = FALSE, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: RevokeUserToken :one
UPDATE users SET min_token_issued_time = $2 WHERE id = $1 RETURNING *;

-- name: CleanTokenIat :one
UPDATE users SET min_token_issued_time = NULL WHERE id = $1 RETURNING *;

-- name: RevokeUsersTokens :one
UPDATE users SET min_token_issued_time = $1 RETURNING *;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: ListUsersByOrganization :many
SELECT * FROM users
WHERE organization_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListUsersByGroup :many
SELECT users.* FROM users
JOIN user_groups ON users.id = user_groups.user_id
WHERE user_groups.group_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

