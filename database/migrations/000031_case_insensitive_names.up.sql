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

-- We want case-insensitive-but-preserving names, in general.  The first step
-- is to replace UNIQUE indexes on names with UNIQUE indexes on lower(name).

CREATE UNIQUE INDEX project_name_lower_idx ON projects (lower(name)) WHERE parent_id IS NULL;
DROP INDEX IF EXISTS projects_name_idx;
CREATE UNIQUE INDEX projects_parent_id_name_lower_idx ON projects (parent_id, lower(name)) WHERE parent_id IS NOT NULL;
DROP INDEX IF EXISTS projects_parent_id_name_idx;

CREATE UNIQUE INDEX providers_project_name_lower_idx ON providers (project_id, lower(name));
DROP INDEX IF EXISTS provider_name_project_id_idx;

CREATE UNIQUE INDEX profiles_project_name_lower_idx ON profiles (project_id, lower(name));
DROP INDEX IF EXISTS profiles_project_id_name_idx;

CREATE UNIQUE INDEX rule_type_project_name_idx ON rule_type (project_id, lower(name));
-- We have an existing index on (provider, project_id, name) that we're not
-- going to remove yet, but we want detach rules from specific providers in
-- a future PR, and will delete that index then.

-- artifact_name_lower_idx already exists and enforces unique lowercase names
-- on artifacts for a given repo.

CREATE UNIQUE INDEX rule_evaluations_results_name_lower_idx ON rule_evaluations
    (
        profile_id, lower(rule_name), repository_id, 
        COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
        entity, rule_type_id,
        COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID)
    ) NULLS NOT DISTINCT;
DROP INDEX IF EXISTS rule_evaluations_results_idx;

-- features name is a primary key, add a unique index on lower(name), other tables
-- have a foreign key to features, so we need to keep the primary key on name
CREATE UNIQUE INDEX features_name_lower_idx ON features (lower(name));
DROP INDEX features_name_idx;

CREATE UNIQUE INDEX bundles_namespace_name_lower_idx ON bundles (lower(namespace), lower(name));