// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: rule_types.sql

package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

const createRuleType = `-- name: CreateRuleType :one
INSERT INTO rule_type (
    name,
    provider,
    project_id,
    description,
    guidance,
    definition,
    severity_value,
    provider_id,
    subscription_id,
    display_name
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6::jsonb,
    $7,
    $8,
    $9,
    $10
) RETURNING id, name, provider, project_id, description, guidance, definition, created_at, updated_at, severity_value, provider_id, subscription_id, display_name
`

type CreateRuleTypeParams struct {
	Name           string          `json:"name"`
	Provider       string          `json:"provider"`
	ProjectID      uuid.UUID       `json:"project_id"`
	Description    string          `json:"description"`
	Guidance       string          `json:"guidance"`
	Definition     json.RawMessage `json:"definition"`
	SeverityValue  Severity        `json:"severity_value"`
	ProviderID     uuid.UUID       `json:"provider_id"`
	SubscriptionID uuid.NullUUID   `json:"subscription_id"`
	DisplayName    string          `json:"display_name"`
}

func (q *Queries) CreateRuleType(ctx context.Context, arg CreateRuleTypeParams) (RuleType, error) {
	row := q.db.QueryRowContext(ctx, createRuleType,
		arg.Name,
		arg.Provider,
		arg.ProjectID,
		arg.Description,
		arg.Guidance,
		arg.Definition,
		arg.SeverityValue,
		arg.ProviderID,
		arg.SubscriptionID,
		arg.DisplayName,
	)
	var i RuleType
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Provider,
		&i.ProjectID,
		&i.Description,
		&i.Guidance,
		&i.Definition,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.SeverityValue,
		&i.ProviderID,
		&i.SubscriptionID,
		&i.DisplayName,
	)
	return i, err
}

const deleteRuleType = `-- name: DeleteRuleType :exec
DELETE FROM rule_type WHERE id = $1
`

func (q *Queries) DeleteRuleType(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteRuleType, id)
	return err
}

const getRuleTypeByID = `-- name: GetRuleTypeByID :one
SELECT id, name, provider, project_id, description, guidance, definition, created_at, updated_at, severity_value, provider_id, subscription_id, display_name FROM rule_type WHERE id = $1
`

func (q *Queries) GetRuleTypeByID(ctx context.Context, id uuid.UUID) (RuleType, error) {
	row := q.db.QueryRowContext(ctx, getRuleTypeByID, id)
	var i RuleType
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Provider,
		&i.ProjectID,
		&i.Description,
		&i.Guidance,
		&i.Definition,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.SeverityValue,
		&i.ProviderID,
		&i.SubscriptionID,
		&i.DisplayName,
	)
	return i, err
}

const getRuleTypeByName = `-- name: GetRuleTypeByName :one
SELECT id, name, provider, project_id, description, guidance, definition, created_at, updated_at, severity_value, provider_id, subscription_id, display_name FROM rule_type WHERE  project_id = $1 AND lower(name) = lower($2)
`

type GetRuleTypeByNameParams struct {
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
}

func (q *Queries) GetRuleTypeByName(ctx context.Context, arg GetRuleTypeByNameParams) (RuleType, error) {
	row := q.db.QueryRowContext(ctx, getRuleTypeByName, arg.ProjectID, arg.Name)
	var i RuleType
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Provider,
		&i.ProjectID,
		&i.Description,
		&i.Guidance,
		&i.Definition,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.SeverityValue,
		&i.ProviderID,
		&i.SubscriptionID,
		&i.DisplayName,
	)
	return i, err
}

const listRuleTypesByProviderAndProject = `-- name: ListRuleTypesByProviderAndProject :many
SELECT id, name, provider, project_id, description, guidance, definition, created_at, updated_at, severity_value, provider_id, subscription_id, display_name FROM rule_type WHERE provider = $1 AND project_id = $2
`

type ListRuleTypesByProviderAndProjectParams struct {
	Provider  string    `json:"provider"`
	ProjectID uuid.UUID `json:"project_id"`
}

func (q *Queries) ListRuleTypesByProviderAndProject(ctx context.Context, arg ListRuleTypesByProviderAndProjectParams) ([]RuleType, error) {
	rows, err := q.db.QueryContext(ctx, listRuleTypesByProviderAndProject, arg.Provider, arg.ProjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RuleType{}
	for rows.Next() {
		var i RuleType
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Provider,
			&i.ProjectID,
			&i.Description,
			&i.Guidance,
			&i.Definition,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.SeverityValue,
			&i.ProviderID,
			&i.SubscriptionID,
			&i.DisplayName,
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

const updateRuleType = `-- name: UpdateRuleType :one
UPDATE rule_type
    SET description = $2, definition = $3::jsonb, severity_value = $4, display_name = $5
    WHERE id = $1
    RETURNING id, name, provider, project_id, description, guidance, definition, created_at, updated_at, severity_value, provider_id, subscription_id, display_name
`

type UpdateRuleTypeParams struct {
	ID            uuid.UUID       `json:"id"`
	Description   string          `json:"description"`
	Definition    json.RawMessage `json:"definition"`
	SeverityValue Severity        `json:"severity_value"`
	DisplayName   string          `json:"display_name"`
}

func (q *Queries) UpdateRuleType(ctx context.Context, arg UpdateRuleTypeParams) (RuleType, error) {
	row := q.db.QueryRowContext(ctx, updateRuleType,
		arg.ID,
		arg.Description,
		arg.Definition,
		arg.SeverityValue,
		arg.DisplayName,
	)
	var i RuleType
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Provider,
		&i.ProjectID,
		&i.Description,
		&i.Guidance,
		&i.Definition,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.SeverityValue,
		&i.ProviderID,
		&i.SubscriptionID,
		&i.DisplayName,
	)
	return i, err
}
