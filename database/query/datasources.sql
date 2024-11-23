-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Data Sources Queries

-- name: CreateDataSource :one
INSERT INTO data_sources (
    name, project_id,display_name
) VALUES (
    $1, $2, sqlc.arg(display_name)
) RETURNING *;

-- name: ListDataSourcesByProject :many
SELECT * FROM data_sources WHERE project_id = $1;

-- name: GetDataSourceByID :one
SELECT * FROM data_sources WHERE id = $1;

-- name: GetDataSourceByName :one
SELECT * FROM data_sources WHERE  project_id = ANY(sqlc.arg(projects)::uuid[]) AND lower(name) = lower(sqlc.arg(name));

-- name: DeleteDataSource :exec
DELETE FROM data_sources WHERE id = $1;

-- name: UpdateDataSource :one
UPDATE data_sources
    SET name = $2, display_name = sqlc.arg(display_name), updated_at = NOW()
    WHERE id = $1
    RETURNING *;


-- Data Source Functions Queries

-- name: CreateDataSourceFunction :one
INSERT INTO data_sources_functions (
    name, type, data_source_id, definition
) VALUES (
    $1, $2, sqlc.arg(display_name), sqlc.arg(definition)::jsonb
) RETURNING *;

-- name: GetDataSourceFunctions :many
SELECT * FROM data_sources_functions WHERE data_source_id = $1;

-- name: DeleteDataSourceFunction :exec
DELETE FROM data_sources_functions WHERE id = $1;

-- name: UpdateDataSourceFunction :one
UPDATE data_sources_functions
    SET name = $2, type = $3, definition = sqlc.arg(definition)::jsonb, updated_at = NOW()
    WHERE id = $1
    RETURNING *;
