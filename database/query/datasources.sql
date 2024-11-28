-- CreateDataSource creates a new datasource in a given project.

-- name: CreateDataSource :one
INSERT INTO data_sources (project_id, name, display_name)
VALUES ($1, $2, $3) RETURNING *;

-- AddDataSourceFunction adds a function to a datasource.

-- name: AddDataSourceFunction :one
INSERT INTO data_sources_functions (data_source_id, project_id, name, type, definition)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- UpdateDataSource updates a datasource in a given project.

-- name: UpdateDataSource :one
UPDATE data_sources
SET display_name = $3
WHERE id = $1 AND project_id = $2
RETURNING *;

-- UpdateDataSourceFunction updates a function in a datasource. We're
-- only able to update the type and definition of the function.

-- name: UpdateDataSourceFunction :one
UPDATE data_sources_functions
SET type = $3, definition = $4, updated_at = NOW()
WHERE data_source_id = $1 AND project_id = $5 AND name = $2
RETURNING *;

-- name: DeleteDataSource :one
DELETE FROM data_sources
WHERE id = $1 AND project_id = $2
RETURNING *;

-- name: DeleteDataSourceFunction :one
DELETE FROM data_sources_functions
WHERE data_source_id = $1 AND name = $2 AND project_id = $3
RETURNING *;

-- DeleteDataSourceFunctions deletes all functions associated with a given datasource
-- in a specific project.

-- name: DeleteDataSourceFunctions :many
DELETE FROM data_sources_functions
WHERE data_source_id = $1 AND project_id = $2
RETURNING *;

-- GetDataSource retrieves a datasource by its id and a project hierarchy.
--
-- Note that to get a datasource for a given project, one can simply
-- pass one project id in the project_id array.

-- name: GetDataSource :one
SELECT * FROM data_sources
WHERE id = $1 AND project_id = ANY(sqlc.arg(projects)::uuid[]);

-- GetDataSourceByName retrieves a datasource by its name and
-- a project hierarchy.
--
-- Note that to get a datasource for a given project, one can simply
-- pass one project id in the project_id array.

-- name: GetDataSourceByName :one
SELECT * FROM data_sources
WHERE name = $1 AND project_id = ANY(sqlc.arg(projects)::uuid[]);

-- ListDataSources retrieves all datasources for project hierarchy.
--
-- Note that to get a datasource for a given project, one can simply
-- pass one project id in the project_id array.

-- name: ListDataSources :many
SELECT * FROM data_sources
WHERE project_id = ANY(sqlc.arg(projects)::uuid[]);

-- ListDataSourceFunctions retrieves all functions for a datasource.

-- name: ListDataSourceFunctions :many
SELECT * FROM data_sources_functions
WHERE data_source_id = $1 AND project_id = $2;

-- ListRuleTypesReferencesByDataSource retrieves all rule types
-- referencing a given data source in a given project.
--
-- name: ListRuleTypesReferencesByDataSource :many
SELECT * FROM rule_type_data_sources
WHERE data_sources_id = $1 AND project_id = $2;

-- AddRuleTypeDataSourceReference adds a link between one rule type
-- and one data source it uses.
--
-- name: AddRuleTypeDataSourceReference :one
INSERT INTO rule_type_data_sources (rule_type_id, data_sources_id, project_id)
VALUES (sqlc.arg(ruleTypeID)::uuid, sqlc.arg(dataSourceID)::uuid, sqlc.arg(projectID)::uuid)
RETURNING rule_type_id, data_sources_id, project_id;
