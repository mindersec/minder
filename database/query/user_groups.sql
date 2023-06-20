-- name: AddUserGroup :one
INSERT INTO user_groups (
  user_id,
  group_id
    ) VALUES (
        $1, $2
) RETURNING *;