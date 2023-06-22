-- name: AddUserRole :one
INSERT INTO user_roles (
  user_id,
  role_id
    ) VALUES (
        $1, $2
) RETURNING *;

-- name: ListUsersByRoleId :many
SELECT user_id FROM user_roles WHERE role_id = $1;

-- name: GetUserRoles :many
SELECT * FROM roles INNER JOIN user_roles ON roles.id = user_roles.role_id WHERE user_roles.user_id = $1;

