// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1
// source: policy_types.sql

package db

import (
	"context"
	"database/sql"
	"time"
)

const deletePolicyType = `-- name: DeletePolicyType :exec
DELETE FROM policy_types WHERE policy_type = $1
`

func (q *Queries) DeletePolicyType(ctx context.Context, policyType string) error {
	_, err := q.db.ExecContext(ctx, deletePolicyType, policyType)
	return err
}

const getPolicyType = `-- name: GetPolicyType :one
SELECT id, provider, policy_type, description, version, created_at, updated_at FROM policy_types WHERE provider = $1 AND policy_type = $2
`

type GetPolicyTypeParams struct {
	Provider   string `json:"provider"`
	PolicyType string `json:"policy_type"`
}

func (q *Queries) GetPolicyType(ctx context.Context, arg GetPolicyTypeParams) (PolicyType, error) {
	row := q.db.QueryRowContext(ctx, getPolicyType, arg.Provider, arg.PolicyType)
	var i PolicyType
	err := row.Scan(
		&i.ID,
		&i.Provider,
		&i.PolicyType,
		&i.Description,
		&i.Version,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getPolicyTypeById = `-- name: GetPolicyTypeById :one
SELECT id, policy_type, description, version, created_at, updated_at FROM policy_types WHERE id = $1
`

type GetPolicyTypeByIdRow struct {
	ID          int32          `json:"id"`
	PolicyType  string         `json:"policy_type"`
	Description sql.NullString `json:"description"`
	Version     string         `json:"version"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (q *Queries) GetPolicyTypeById(ctx context.Context, id int32) (GetPolicyTypeByIdRow, error) {
	row := q.db.QueryRowContext(ctx, getPolicyTypeById, id)
	var i GetPolicyTypeByIdRow
	err := row.Scan(
		&i.ID,
		&i.PolicyType,
		&i.Description,
		&i.Version,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getPolicyTypes = `-- name: GetPolicyTypes :many
SELECT id, provider, policy_type, description, version, created_at, updated_at FROM policy_types WHERE provider = $1 ORDER BY policy_type
`

func (q *Queries) GetPolicyTypes(ctx context.Context, provider string) ([]PolicyType, error) {
	rows, err := q.db.QueryContext(ctx, getPolicyTypes, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PolicyType{}
	for rows.Next() {
		var i PolicyType
		if err := rows.Scan(
			&i.ID,
			&i.Provider,
			&i.PolicyType,
			&i.Description,
			&i.Version,
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
