-- Copyright 2024 Stacklok, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

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
DROP INDEX artifact_name_lower_idx;

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

DROP INDEX entity_execution_lock_idx;
DROP INDEX flush_cache_idx;

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