-- GetFeatureInProject verifies if a feature is available for a specific project.
-- It returns the settings for the feature if it is available.

-- name: GetFeatureInProject :one
SELECT f.settings FROM entitlements e
INNER JOIN features f ON f.name = e.feature
WHERE e.project_id = sqlc.arg(project_id)::UUID AND e.feature = sqlc.arg(feature)::TEXT;

-- name: GetEntitlementFeaturesByProjectID :many
SELECT feature
FROM entitlements
WHERE project_id = sqlc.arg(project_id)::UUID;

-- name: CreateEntitlements :exec
INSERT INTO entitlements (feature, project_id)
SELECT unnest(sqlc.arg(features)::text[]), sqlc.arg(project_id)::UUID
ON CONFLICT DO NOTHING;
