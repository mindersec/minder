// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: projects.sql

package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const createProject = `-- name: CreateProject :one
INSERT INTO projects (
    name,
    parent_id,
    metadata
) VALUES (
    $1, $2, $3::jsonb
) RETURNING id, name, is_organization, metadata, parent_id, created_at, updated_at
`

type CreateProjectParams struct {
	Name     string          `json:"name"`
	ParentID uuid.NullUUID   `json:"parent_id"`
	Metadata json.RawMessage `json:"metadata"`
}

func (q *Queries) CreateProject(ctx context.Context, arg CreateProjectParams) (Project, error) {
	row := q.db.QueryRowContext(ctx, createProject, arg.Name, arg.ParentID, arg.Metadata)
	var i Project
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.IsOrganization,
		&i.Metadata,
		&i.ParentID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createProjectWithID = `-- name: CreateProjectWithID :one
INSERT INTO projects (
    id,
    name,
    metadata
) VALUES (
    $1, $2, $3::jsonb
) RETURNING id, name, is_organization, metadata, parent_id, created_at, updated_at
`

type CreateProjectWithIDParams struct {
	ID       uuid.UUID       `json:"id"`
	Name     string          `json:"name"`
	Metadata json.RawMessage `json:"metadata"`
}

func (q *Queries) CreateProjectWithID(ctx context.Context, arg CreateProjectWithIDParams) (Project, error) {
	row := q.db.QueryRowContext(ctx, createProjectWithID, arg.ID, arg.Name, arg.Metadata)
	var i Project
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.IsOrganization,
		&i.Metadata,
		&i.ParentID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteProject = `-- name: DeleteProject :many
WITH RECURSIVE get_children AS (
    SELECT id, parent_id FROM projects
    WHERE projects.id = $1

    UNION

    SELECT p.id, p.parent_id FROM projects p
    INNER JOIN get_children gc ON p.parent_id = gc.id
)
DELETE FROM projects
WHERE id IN (SELECT id FROM get_children)
RETURNING id, name, metadata, created_at, updated_at, parent_id
`

type DeleteProjectRow struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	ParentID  uuid.NullUUID   `json:"parent_id"`
}

func (q *Queries) DeleteProject(ctx context.Context, id uuid.UUID) ([]DeleteProjectRow, error) {
	rows, err := q.db.QueryContext(ctx, deleteProject, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DeleteProjectRow{}
	for rows.Next() {
		var i DeleteProjectRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Metadata,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.ParentID,
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

const getChildrenProjects = `-- name: GetChildrenProjects :many
WITH RECURSIVE get_children AS (
    SELECT projects.id, projects.name, projects.metadata, projects.parent_id, projects.created_at, projects.updated_at FROM projects 
    WHERE projects.id = $1

    UNION

    (
        SELECT p.id, p.name, p.metadata, p.parent_id, p.created_at, p.updated_at FROM projects p
        INNER JOIN get_children gc ON p.parent_id = gc.id
        ORDER BY p.created_at ASC
    )
)
SELECT id, name, metadata, parent_id, created_at, updated_at FROM get_children
`

type GetChildrenProjectsRow struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Metadata  json.RawMessage `json:"metadata"`
	ParentID  uuid.NullUUID   `json:"parent_id"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (q *Queries) GetChildrenProjects(ctx context.Context, id uuid.UUID) ([]GetChildrenProjectsRow, error) {
	rows, err := q.db.QueryContext(ctx, getChildrenProjects, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetChildrenProjectsRow{}
	for rows.Next() {
		var i GetChildrenProjectsRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Metadata,
			&i.ParentID,
			&i.CreatedAt,
			&i.UpdatedAt,
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

const getImmediateChildrenProjects = `-- name: GetImmediateChildrenProjects :many


SELECT id, name, is_organization, metadata, parent_id, created_at, updated_at FROM projects
WHERE parent_id = $1::UUID
`

// GetImmediateChildrenProjects is a query that returns all the immediate children of a project.
func (q *Queries) GetImmediateChildrenProjects(ctx context.Context, parentID uuid.UUID) ([]Project, error) {
	rows, err := q.db.QueryContext(ctx, getImmediateChildrenProjects, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Project{}
	for rows.Next() {
		var i Project
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.IsOrganization,
			&i.Metadata,
			&i.ParentID,
			&i.CreatedAt,
			&i.UpdatedAt,
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

const getParentProjects = `-- name: GetParentProjects :many
WITH RECURSIVE get_parents AS (
    SELECT id, parent_id, created_at FROM projects 
    WHERE projects.id = $1

    UNION

    (
        SELECT p.id, p.parent_id, p.created_at FROM projects p
        INNER JOIN get_parents gp ON p.id = gp.parent_id
        ORDER BY p.created_at ASC
    )
)
SELECT id FROM get_parents
`

func (q *Queries) GetParentProjects(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.db.QueryContext(ctx, getParentProjects, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getParentProjectsUntil = `-- name: GetParentProjectsUntil :many
WITH RECURSIVE get_parents_until AS (
    SELECT id, parent_id, created_at FROM projects 
    WHERE projects.id = $1

    UNION

    (
        SELECT p.id, p.parent_id, p.created_at FROM projects p
        INNER JOIN get_parents_until gpu ON p.id = gpu.parent_id
        WHERE p.id != $2
        ORDER BY p.created_at ASC
    )
)
SELECT id FROM get_parents_until
`

type GetParentProjectsUntilParams struct {
	ID   uuid.UUID `json:"id"`
	ID_2 uuid.UUID `json:"id_2"`
}

func (q *Queries) GetParentProjectsUntil(ctx context.Context, arg GetParentProjectsUntilParams) ([]uuid.UUID, error) {
	rows, err := q.db.QueryContext(ctx, getParentProjectsUntil, arg.ID, arg.ID_2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProjectByID = `-- name: GetProjectByID :one
SELECT id, name, is_organization, metadata, parent_id, created_at, updated_at FROM projects
WHERE id = $1 AND is_organization = FALSE LIMIT 1
`

func (q *Queries) GetProjectByID(ctx context.Context, id uuid.UUID) (Project, error) {
	row := q.db.QueryRowContext(ctx, getProjectByID, id)
	var i Project
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.IsOrganization,
		&i.Metadata,
		&i.ParentID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getProjectByName = `-- name: GetProjectByName :one
SELECT id, name, is_organization, metadata, parent_id, created_at, updated_at FROM projects
WHERE lower(name) = lower($1) AND is_organization = FALSE LIMIT 1
`

func (q *Queries) GetProjectByName(ctx context.Context, name string) (Project, error) {
	row := q.db.QueryRowContext(ctx, getProjectByName, name)
	var i Project
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.IsOrganization,
		&i.Metadata,
		&i.ParentID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listNonOrgProjects = `-- name: ListNonOrgProjects :many

SELECT id, name, is_organization, metadata, parent_id, created_at, updated_at FROM projects
WHERE is_organization = FALSE
`

// ListNonOrgProjects is a query that lists all non-organization projects.
// projects have a boolean field is_organization that is set to true if the project is an organization.
// this flag is no longer used and will be removed in the future.
func (q *Queries) ListNonOrgProjects(ctx context.Context) ([]Project, error) {
	rows, err := q.db.QueryContext(ctx, listNonOrgProjects)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Project{}
	for rows.Next() {
		var i Project
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.IsOrganization,
			&i.Metadata,
			&i.ParentID,
			&i.CreatedAt,
			&i.UpdatedAt,
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

const listOldOrgProjects = `-- name: ListOldOrgProjects :many

SELECT id, name, is_organization, metadata, parent_id, created_at, updated_at FROM projects
WHERE is_organization = TRUE
`

// ListOrgProjects is a query that lists all organization projects.
// projects have a boolean field is_organization that is set to true if the project is an organization.
// this flag is no longer used and will be removed in the future.
func (q *Queries) ListOldOrgProjects(ctx context.Context) ([]Project, error) {
	rows, err := q.db.QueryContext(ctx, listOldOrgProjects)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Project{}
	for rows.Next() {
		var i Project
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.IsOrganization,
			&i.Metadata,
			&i.ParentID,
			&i.CreatedAt,
			&i.UpdatedAt,
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

const orphanProject = `-- name: OrphanProject :one

UPDATE projects
SET metadata = $2, parent_id = NULL
WHERE id = $1 RETURNING id, name, is_organization, metadata, parent_id, created_at, updated_at
`

type OrphanProjectParams struct {
	ID       uuid.UUID       `json:"id"`
	Metadata json.RawMessage `json:"metadata"`
}

// OrphanProject is a query that sets the parent_id of a project to NULL.
func (q *Queries) OrphanProject(ctx context.Context, arg OrphanProjectParams) (Project, error) {
	row := q.db.QueryRowContext(ctx, orphanProject, arg.ID, arg.Metadata)
	var i Project
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.IsOrganization,
		&i.Metadata,
		&i.ParentID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateProjectMeta = `-- name: UpdateProjectMeta :one
UPDATE projects
SET metadata = $2
WHERE id = $1 RETURNING id, name, is_organization, metadata, parent_id, created_at, updated_at
`

type UpdateProjectMetaParams struct {
	ID       uuid.UUID       `json:"id"`
	Metadata json.RawMessage `json:"metadata"`
}

func (q *Queries) UpdateProjectMeta(ctx context.Context, arg UpdateProjectMetaParams) (Project, error) {
	row := q.db.QueryRowContext(ctx, updateProjectMeta, arg.ID, arg.Metadata)
	var i Project
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.IsOrganization,
		&i.Metadata,
		&i.ParentID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
