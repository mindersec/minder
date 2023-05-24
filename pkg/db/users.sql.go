// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0
// source: users.sql

package db

import (
	"context"
	"database/sql"
)

const createUser = `-- name: CreateUser :one
INSERT INTO users (organisation_id, group_id, email, username, password, first_name, last_name) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, organisation_id, group_id, role_id, email, username, password, first_name, last_name, created_at, updated_at
`

type CreateUserParams struct {
	OrganisationID sql.NullInt32 `json:"organisation_id"`
	GroupID        sql.NullInt32 `json:"group_id"`
	Email          string        `json:"email"`
	Username       string        `json:"username"`
	Password       string        `json:"password"`
	FirstName      string        `json:"first_name"`
	LastName       string        `json:"last_name"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, createUser,
		arg.OrganisationID,
		arg.GroupID,
		arg.Email,
		arg.Username,
		arg.Password,
		arg.FirstName,
		arg.LastName,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganisationID,
		&i.GroupID,
		&i.RoleID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteUser = `-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1
`

func (q *Queries) DeleteUser(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, deleteUser, id)
	return err
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT id, organisation_id, group_id, role_id, email, username, password, first_name, last_name, created_at, updated_at FROM users WHERE email = $1
`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByEmail, email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganisationID,
		&i.GroupID,
		&i.RoleID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserByID = `-- name: GetUserByID :one
SELECT id, organisation_id, group_id, role_id, email, username, password, first_name, last_name, created_at, updated_at FROM users WHERE id = $1
`

func (q *Queries) GetUserByID(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByID, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganisationID,
		&i.GroupID,
		&i.RoleID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserByUserName = `-- name: GetUserByUserName :one
SELECT id, organisation_id, group_id, role_id, email, username, password, first_name, last_name, created_at, updated_at FROM users WHERE username = $1
`

func (q *Queries) GetUserByUserName(ctx context.Context, username string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByUserName, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganisationID,
		&i.GroupID,
		&i.RoleID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listUsers = `-- name: ListUsers :many
SELECT id, organisation_id, group_id, role_id, email, username, password, first_name, last_name, created_at, updated_at FROM users
`

func (q *Queries) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, listUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []User{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.OrganisationID,
			&i.GroupID,
			&i.RoleID,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.FirstName,
			&i.LastName,
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

const updateUser = `-- name: UpdateUser :one
UPDATE users SET organisation_id = $2, group_id = $3, email = $4, username = $5, password = $6, first_name = $7, last_name = $8, updated_at = NOW() WHERE id = $1 RETURNING id, organisation_id, group_id, role_id, email, username, password, first_name, last_name, created_at, updated_at
`

type UpdateUserParams struct {
	ID             int32         `json:"id"`
	OrganisationID sql.NullInt32 `json:"organisation_id"`
	GroupID        sql.NullInt32 `json:"group_id"`
	Email          string        `json:"email"`
	Username       string        `json:"username"`
	Password       string        `json:"password"`
	FirstName      string        `json:"first_name"`
	LastName       string        `json:"last_name"`
}

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, updateUser,
		arg.ID,
		arg.OrganisationID,
		arg.GroupID,
		arg.Email,
		arg.Username,
		arg.Password,
		arg.FirstName,
		arg.LastName,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganisationID,
		&i.GroupID,
		&i.RoleID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
