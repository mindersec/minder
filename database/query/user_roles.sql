-- name: AddUserRole :one
INSERT INTO user_roles (
  user_id,
  role_id
    ) VALUES (
        $1, $2
) RETURNING *;

-- name: ListUsersByRoleId :many
SELECT user_id FROM user_roles WHERE role_id = $1;
