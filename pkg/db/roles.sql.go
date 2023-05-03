// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.17.2
// source: roles.sql

package db

import (
	"context"
)

const createRole = `-- name: CreateRole :one
INSERT INTO roles (organisation_id, name) VALUES ($1, $2) RETURNING id, organisation_id, name, created_at, updated_at
`

type CreateRoleParams struct {
	OrganisationID int32  `json:"organisation_id"`
	Name           string `json:"name"`
}

func (q *Queries) CreateRole(ctx context.Context, arg CreateRoleParams) (Role, error) {
	row := q.db.QueryRowContext(ctx, createRole, arg.OrganisationID, arg.Name)
	var i Role
	err := row.Scan(
		&i.ID,
		&i.OrganisationID,
		&i.Name,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteRole = `-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1
`

func (q *Queries) DeleteRole(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, deleteRole, id)
	return err
}

const getRoleByID = `-- name: GetRoleByID :one
SELECT id, organisation_id, name, created_at, updated_at FROM roles WHERE id = $1
`

func (q *Queries) GetRoleByID(ctx context.Context, id int32) (Role, error) {
	row := q.db.QueryRowContext(ctx, getRoleByID, id)
	var i Role
	err := row.Scan(
		&i.ID,
		&i.OrganisationID,
		&i.Name,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listRoles = `-- name: ListRoles :many
SELECT id, organisation_id, name, created_at, updated_at FROM roles
`

func (q *Queries) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := q.db.QueryContext(ctx, listRoles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Role{}
	for rows.Next() {
		var i Role
		if err := rows.Scan(
			&i.ID,
			&i.OrganisationID,
			&i.Name,
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

const updateRole = `-- name: UpdateRole :one
UPDATE roles SET name = $2, updated_at = NOW() WHERE id = $1 RETURNING id, organisation_id, name, created_at, updated_at
`

type UpdateRoleParams struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

func (q *Queries) UpdateRole(ctx context.Context, arg UpdateRoleParams) (Role, error) {
	row := q.db.QueryRowContext(ctx, updateRole, arg.ID, arg.Name)
	var i Role
	err := row.Scan(
		&i.ID,
		&i.OrganisationID,
		&i.Name,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
