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

DROP INDEX bundles_namespace_name_lower_idx;

CREATE UNIQUE INDEX IF NOT EXISTS features_name_idx ON features (name);
DROP INDEX features_name_lower_idx;

CREATE UNIQUE INDEX IF NOT EXISTS rule_evaluations_results_idx
  ON rule_evaluations (profile_id, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
                       entity, rule_type_id, COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID),
                       rule_name) NULLS NOT DISTINCT;
DROP INDEX rule_evaluations_results_name_lower_idx;

DROP INDEX rule_type_project_name_idx;

CREATE UNIQUE INDEX IF NOT EXISTS profiles_project_id_name_idx ON profiles (project_id, name);
DROP INDEX profiles_project_name_lower_idx;

CREATE UNIQUE INDEX IF NOT EXISTS provider_name_project_id_idx ON providers (name, project_id);
DROP INDEX providers_project_name_lower_idx;

CREATE UNIQUE INDEX IF NOT EXISTS projects_parent_id_name_idx ON projects (parent_id, name) WHERE parent_id IS NOT NULL;
DROP INDEX projects_parent_id_name_lower_idx;
CREATE UNIQUE INDEX IF NOT EXISTS projects_name_idx ON projects (name) WHERE parent_id IS NULL;
DROP INDEX project_name_lower_idx;
