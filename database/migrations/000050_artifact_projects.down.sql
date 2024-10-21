-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Artifact changes

-- make repository_id not nullable in artifacts
ALTER TABLE artifacts ALTER COLUMN repository_id SET NOT NULL;

-- remove foreign key constraints
ALTER TABLE artifacts DROP CONSTRAINT fk_artifacts_project_id;
ALTER TABLE artifacts DROP CONSTRAINT fk_artifacts_provider_id_and_name;

-- remove project_id, provider_id and provider_name columns from artifacts table
ALTER TABLE artifacts DROP COLUMN project_id;
ALTER TABLE artifacts DROP COLUMN provider_id;
ALTER TABLE artifacts DROP COLUMN provider_name;

-- recreate index artifact_name_lower_idx on artifacts but without project_id
DROP INDEX IF EXISTS artifact_name_lower_idx;

CREATE INDEX artifact_name_lower_idx ON artifacts (repository_id, LOWER(artifact_name));

COMMIT;

BEGIN;

-- Entity Execution lock changes

-- make repository_id not nullable in entity_execution_lock and flush_cache
ALTER TABLE entity_execution_lock ALTER COLUMN repository_id SET NOT NULL;
ALTER TABLE flush_cache ALTER COLUMN repository_id SET NOT NULL;

-- remove foreign key constraints
ALTER TABLE entity_execution_lock DROP CONSTRAINT fk_entity_execution_lock_project_id;

-- remove project_id column from entity_execution_lock
ALTER TABLE entity_execution_lock DROP COLUMN project_id;

-- remove project_id column from flush_cache
ALTER TABLE flush_cache DROP COLUMN project_id;

COMMIT;

BEGIN;

DROP INDEX IF EXISTS entity_execution_lock_idx;
DROP INDEX IF EXISTS flush_cache_idx;

-- recreate entity_execution_lock_idx and flush_cache_idx indexes with nullable repository_id
CREATE UNIQUE INDEX IF NOT EXISTS entity_execution_lock_idx ON entity_execution_lock(
    entity,
    repository_id,
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID));

CREATE UNIQUE INDEX IF NOT EXISTS flush_cache_idx ON flush_cache(
    entity,
    repository_id,
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID));

COMMIT;
