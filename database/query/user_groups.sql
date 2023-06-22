-- name: AddUserGroup :one
INSERT INTO user_groups (
  user_id,
  group_id
    ) VALUES (
        $1, $2
) RETURNING *;

-- name: GetUserGroups :many
SELECT * FROM groups INNER JOIN user_groups ON groups.id = user_groups.group_id WHERE user_groups.user_id = $1;