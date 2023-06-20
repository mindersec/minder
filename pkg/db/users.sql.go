// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0
// source: users.sql

package db

import (
	"context"
	"database/sql"
)

const cleanTokenIat = `-- name: CleanTokenIat :one
UPDATE users SET min_token_issued_time = NULL WHERE id = $1 RETURNING id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time
`

func (q *Queries) CleanTokenIat(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRowContext(ctx, cleanTokenIat, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
	)
	return i, err
}

const createUser = `-- name: CreateUser :one
INSERT INTO users (organization_id, email, username, password, first_name, last_name, is_protected, needs_password_change) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time
`

type CreateUserParams struct {
	OrganizationID      int32          `json:"organization_id"`
	Email               sql.NullString `json:"email"`
	Username            string         `json:"username"`
	Password            string         `json:"password"`
	FirstName           sql.NullString `json:"first_name"`
	LastName            sql.NullString `json:"last_name"`
	IsProtected         bool           `json:"is_protected"`
	NeedsPasswordChange bool           `json:"needs_password_change"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, createUser,
		arg.OrganizationID,
		arg.Email,
		arg.Username,
		arg.Password,
		arg.FirstName,
		arg.LastName,
		arg.IsProtected,
		arg.NeedsPasswordChange,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
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
SELECT id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time FROM users WHERE email = $1
`

func (q *Queries) GetUserByEmail(ctx context.Context, email sql.NullString) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByEmail, email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
	)
	return i, err
}

const getUserByID = `-- name: GetUserByID :one
SELECT id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time FROM users WHERE id = $1
`

func (q *Queries) GetUserByID(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByID, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
	)
	return i, err
}

const getUserByUserName = `-- name: GetUserByUserName :one
SELECT id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time FROM users WHERE username = $1
`

func (q *Queries) GetUserByUserName(ctx context.Context, username string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByUserName, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
	)
	return i, err
}

const getUserClaims = `-- name: GetUserClaims :one
SELECT u.id as user_id, g.organization_id as organization_id, u.min_token_issued_time AS min_token_issued_time,
u.needs_password_change as needs_password_change,
ARRAY_AGG(jsonb_build_object('role_id', roles.id, 'is_admin', roles.is_admin, 'group_id', roles.group_id, 'organization_id', roles.organization_id)) AS role_info,
ARRAY_AGG(g.id) AS group_ids
 FROM users u
INNER JOIN roles r ON u.role_id = r.id INNER JOIN groups g ON r.group_id = g.id WHERE u.id = $1
`

type GetUserClaimsRow struct {
	UserID              int32        `json:"user_id"`
	OrganizationID      int32        `json:"organization_id"`
	MinTokenIssuedTime  sql.NullTime `json:"min_token_issued_time"`
	NeedsPasswordChange bool         `json:"needs_password_change"`
	RoleInfo            interface{}  `json:"role_info"`
	GroupIds            interface{}  `json:"group_ids"`
}

func (q *Queries) GetUserClaims(ctx context.Context, id int32) (GetUserClaimsRow, error) {
	row := q.db.QueryRowContext(ctx, getUserClaims, id)
	var i GetUserClaimsRow
	err := row.Scan(
		&i.UserID,
		&i.OrganizationID,
		&i.MinTokenIssuedTime,
		&i.NeedsPasswordChange,
		&i.RoleInfo,
		&i.GroupIds,
	)
	return i, err
}

const listUsers = `-- name: ListUsers :many
SELECT id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time FROM users
ORDER BY id
LIMIT $1
OFFSET $2
`

type ListUsersParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

func (q *Queries) ListUsers(ctx context.Context, arg ListUsersParams) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, listUsers, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []User{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.OrganizationID,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.NeedsPasswordChange,
			&i.FirstName,
			&i.LastName,
			&i.IsProtected,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.MinTokenIssuedTime,
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

const listUsersByGroup = `-- name: ListUsersByGroup :many
SELECT users.id, users.organization_id, users.email, users.username, users.password, users.needs_password_change, users.first_name, users.last_name, users.is_protected, users.created_at, users.updated_at, users.min_token_issued_time FROM users
JOIN user_groups ON users.id = user_groups.user_id
WHERE user_groups.group_id = $1
ORDER BY id
LIMIT $2
OFFSET $3
`

type ListUsersByGroupParams struct {
	GroupID int32 `json:"group_id"`
	Limit   int32 `json:"limit"`
	Offset  int32 `json:"offset"`
}

func (q *Queries) ListUsersByGroup(ctx context.Context, arg ListUsersByGroupParams) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, listUsersByGroup, arg.GroupID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []User{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.OrganizationID,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.NeedsPasswordChange,
			&i.FirstName,
			&i.LastName,
			&i.IsProtected,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.MinTokenIssuedTime,
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

const listUsersByOrganization = `-- name: ListUsersByOrganization :many
SELECT id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time FROM users
WHERE organization_id = $1
ORDER BY id
LIMIT $2
OFFSET $3
`

type ListUsersByOrganizationParams struct {
	OrganizationID int32 `json:"organization_id"`
	Limit          int32 `json:"limit"`
	Offset         int32 `json:"offset"`
}

func (q *Queries) ListUsersByOrganization(ctx context.Context, arg ListUsersByOrganizationParams) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, listUsersByOrganization, arg.OrganizationID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []User{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.OrganizationID,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.NeedsPasswordChange,
			&i.FirstName,
			&i.LastName,
			&i.IsProtected,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.MinTokenIssuedTime,
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

const revokeUserToken = `-- name: RevokeUserToken :one
UPDATE users SET min_token_issued_time = $2 WHERE id = $1 RETURNING id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time
`

type RevokeUserTokenParams struct {
	ID                 int32        `json:"id"`
	MinTokenIssuedTime sql.NullTime `json:"min_token_issued_time"`
}

func (q *Queries) RevokeUserToken(ctx context.Context, arg RevokeUserTokenParams) (User, error) {
	row := q.db.QueryRowContext(ctx, revokeUserToken, arg.ID, arg.MinTokenIssuedTime)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
	)
	return i, err
}

const revokeUsersTokens = `-- name: RevokeUsersTokens :one
UPDATE users SET min_token_issued_time = $1 RETURNING id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time
`

func (q *Queries) RevokeUsersTokens(ctx context.Context, minTokenIssuedTime sql.NullTime) (User, error) {
	row := q.db.QueryRowContext(ctx, revokeUsersTokens, minTokenIssuedTime)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
	)
	return i, err
}

const updatePassword = `-- name: UpdatePassword :one
UPDATE users SET password = $2, needs_password_change = FALSE, updated_at = NOW() WHERE id = $1 RETURNING id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time
`

type UpdatePasswordParams struct {
	ID       int32  `json:"id"`
	Password string `json:"password"`
}

func (q *Queries) UpdatePassword(ctx context.Context, arg UpdatePasswordParams) (User, error) {
	row := q.db.QueryRowContext(ctx, updatePassword, arg.ID, arg.Password)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
	)
	return i, err
}

const updateUser = `-- name: UpdateUser :one
UPDATE users SET email = $2, username = $3, password = $4, first_name = $5, last_name = $6, is_protected = $7, updated_at = NOW() WHERE id = $1 RETURNING id, organization_id, email, username, password, needs_password_change, first_name, last_name, is_protected, created_at, updated_at, min_token_issued_time
`

type UpdateUserParams struct {
	ID          int32          `json:"id"`
	Email       sql.NullString `json:"email"`
	Username    string         `json:"username"`
	Password    string         `json:"password"`
	FirstName   sql.NullString `json:"first_name"`
	LastName    sql.NullString `json:"last_name"`
	IsProtected bool           `json:"is_protected"`
}

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, updateUser,
		arg.ID,
		arg.Email,
		arg.Username,
		arg.Password,
		arg.FirstName,
		arg.LastName,
		arg.IsProtected,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.OrganizationID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.NeedsPasswordChange,
		&i.FirstName,
		&i.LastName,
		&i.IsProtected,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.MinTokenIssuedTime,
	)
	return i, err
}
