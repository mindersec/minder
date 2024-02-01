-- name: AddUserProject :one
INSERT INTO user_projects (
  user_id,
  project_id
    ) VALUES (
        $1, $2
) ON CONFLICT DO NOTHING RETURNING *;

-- name: RemoveUserProject :one
DELETE FROM user_projects WHERE user_id = $1 AND project_id = $2 RETURNING *;

-- name: GetUserProjects :many
SELECT * FROM projects INNER JOIN user_projects ON projects.id = user_projects.project_id WHERE user_projects.user_id = $1;