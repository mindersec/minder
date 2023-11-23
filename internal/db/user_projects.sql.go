// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0
// source: user_projects.sql

package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const addUserProject = `-- name: AddUserProject :one
INSERT INTO user_projects (
  user_id,
  project_id
    ) VALUES (
        $1, $2
) RETURNING id, user_id, project_id
`

type AddUserProjectParams struct {
	UserID    int32     `json:"user_id"`
	ProjectID uuid.UUID `json:"project_id"`
}

func (q *Queries) AddUserProject(ctx context.Context, arg AddUserProjectParams) (UserProject, error) {
	row := q.db.QueryRowContext(ctx, addUserProject, arg.UserID, arg.ProjectID)
	var i UserProject
	err := row.Scan(&i.ID, &i.UserID, &i.ProjectID)
	return i, err
}

const getUserProjects = `-- name: GetUserProjects :many
SELECT projects.id, name, is_organization, metadata, parent_id, created_at, updated_at, user_projects.id, user_id, project_id FROM projects INNER JOIN user_projects ON projects.id = user_projects.project_id WHERE user_projects.user_id = $1
`

type GetUserProjectsRow struct {
	ID             uuid.UUID       `json:"id"`
	Name           string          `json:"name"`
	IsOrganization bool            `json:"is_organization"`
	Metadata       json.RawMessage `json:"metadata"`
	ParentID       uuid.NullUUID   `json:"parent_id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ID_2           int32           `json:"id_2"`
	UserID         int32           `json:"user_id"`
	ProjectID      uuid.UUID       `json:"project_id"`
}

func (q *Queries) GetUserProjects(ctx context.Context, userID int32) ([]GetUserProjectsRow, error) {
	rows, err := q.db.QueryContext(ctx, getUserProjects, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetUserProjectsRow{}
	for rows.Next() {
		var i GetUserProjectsRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.IsOrganization,
			&i.Metadata,
			&i.ParentID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.ID_2,
			&i.UserID,
			&i.ProjectID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
