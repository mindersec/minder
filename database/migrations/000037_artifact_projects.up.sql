
-- Artifact changes

-- Add project_id column to artifacts table
ALTER TABLE artifacts ADD COLUMN project_id UUID;

-- make it a foreign key to projects
ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_project_id FOREIGN KEY (project_id) REFERENCES projects (id);

-- Add provider_id and provider_name columns to artifacts table
ALTER TABLE artifacts ADD COLUMN provider_id UUID;
ALTER TABLE artifacts ADD COLUMN provider_name TEXT;

-- make provider_id a foreign key to providers
ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_provider_id_and_name FOREIGN KEY (provider_id, provider_name) REFERENCES providers (id, name);

-- remove index artifact_name_lower_idx from artifacts
DROP INDEX artifact_name_lower_idx;

-- recreate index artifact_name_lower_idx on artifacts but with project_id
CREATE INDEX artifact_name_lower_idx ON artifacts (project_id, LOWER(artifact_name));

-- populate project_id, provider_id and provider_name in artifacts
UPDATE artifacts
SET project_id = repositories.project_id,
    provider_id = repositories.provider_id,
    provider_name = repositories.provider
FROM repositories
WHERE artifacts.repository_id = repositories.id;

-- make repository_id nullable in artifacts
ALTER TABLE artifacts ALTER COLUMN repository_id DROP NOT NULL;

-- make project_id not nullable in artifacts
ALTER TABLE artifacts ALTER COLUMN project_id SET NOT NULL;

ALTER TABLE artifacts ALTER COLUMN provider_id SET NOT NULL;

ALTER TABLE artifacts ALTER COLUMN provider_name SET NOT NULL;

-- Now that repository_id's are nullable, let's index artifacts by repository_id where the repository_id is not null
CREATE INDEX artifacts_repository_id_idx ON artifacts (repository_id) WHERE repository_id IS NOT NULL;

-- make repository_id nullable in entity_execution_lock and flush_cache
ALTER TABLE entity_execution_lock ALTER COLUMN repository_id DROP NOT NULL;
ALTER TABLE flush_cache ALTER COLUMN repository_id DROP NOT NULL;

-- Add project_id column to entity_execution_lock table and make it a foreign key to projects
ALTER TABLE entity_execution_lock ADD COLUMN project_id UUID;
ALTER TABLE entity_execution_lock ADD CONSTRAINT fk_entity_execution_lock_project_id FOREIGN KEY (project_id) REFERENCES projects (id);

-- Add project_id column to flush_cache table and make it a foreign key to projects
ALTER TABLE flush_cache ADD COLUMN project_id UUID;
ALTER TABLE flush_cache ADD CONSTRAINT fk_flush_cache_project_id FOREIGN KEY (project_id) REFERENCES projects (id);

-- delete entity_execution_lock_idx and flush_cache_idx indexes.
DROP INDEX entity_execution_lock_idx;
DROP INDEX flush_cache_idx;

-- recreate entity_execution_lock_idx and flush_cache_idx indexes with nullable repository_id
CREATE UNIQUE INDEX IF NOT EXISTS entity_execution_lock_idx ON entity_execution_lock(
    entity,
    COALESCE(repository_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID));

CREATE UNIQUE INDEX IF NOT EXISTS flush_cache_idx ON flush_cache(
    entity,
    COALESCE(repository_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID));