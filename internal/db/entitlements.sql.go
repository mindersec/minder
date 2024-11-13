// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: entitlements.sql

package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

const createEntitlement = `-- name: CreateEntitlement :exec
INSERT INTO entitlements (feature, project_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING
`

type CreateEntitlementParams struct {
	Feature   string    `json:"feature"`
	ProjectID uuid.UUID `json:"project_id"`
}

func (q *Queries) CreateEntitlement(ctx context.Context, arg CreateEntitlementParams) error {
	_, err := q.db.ExecContext(ctx, createEntitlement, arg.Feature, arg.ProjectID)
	return err
}

const getEntitlementFeaturesByProjectID = `-- name: GetEntitlementFeaturesByProjectID :many
SELECT feature
FROM entitlements
WHERE project_id = $1::UUID
`

func (q *Queries) GetEntitlementFeaturesByProjectID(ctx context.Context, projectID uuid.UUID) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getEntitlementFeaturesByProjectID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var feature string
		if err := rows.Scan(&feature); err != nil {
			return nil, err
		}
		items = append(items, feature)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getFeatureInProject = `-- name: GetFeatureInProject :one

SELECT f.settings FROM entitlements e
INNER JOIN features f ON f.name = e.feature
WHERE e.project_id = $1::UUID AND e.feature = $2::TEXT
`

type GetFeatureInProjectParams struct {
	ProjectID uuid.UUID `json:"project_id"`
	Feature   string    `json:"feature"`
}

// GetFeatureInProject verifies if a feature is available for a specific project.
// It returns the settings for the feature if it is available.
func (q *Queries) GetFeatureInProject(ctx context.Context, arg GetFeatureInProjectParams) (json.RawMessage, error) {
	row := q.db.QueryRowContext(ctx, getFeatureInProject, arg.ProjectID, arg.Feature)
	var settings json.RawMessage
	err := row.Scan(&settings)
	return settings, err
}
