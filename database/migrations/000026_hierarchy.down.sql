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

--- Remove the unique constraint to the providers table between name and project_id
ALTER TABLE providers
  DROP CONSTRAINT providers_name_id_key;

--- Remove the repositories_provider_id_name_fkey constraint
--- from the repositories, rule_type and profiles tables
ALTER TABLE repositories
  DROP CONSTRAINT repositories_provider_id_name_fkey;

ALTER TABLE rule_type
    DROP CONSTRAINT rule_type_provider_id_name_fkey;

ALTER TABLE profiles
    DROP CONSTRAINT profiles_provider_id_name_fkey;

--- Remove the provider_id column from the repositories, rule_type and profiles tables
ALTER TABLE repositories
  DROP COLUMN provider_id;

ALTER TABLE rule_type
    DROP COLUMN provider_id;

ALTER TABLE profiles
    DROP COLUMN provider_id;

--- Change the project_id column from being a foreign key from the projects
--- table to being a foreign key from the providers table
ALTER TABLE repositories
  DROP CONSTRAINT repositories_project_id_fkey;

ALTER TABLE repositories
    ADD CONSTRAINT repositories_project_id_provider_fkey
    FOREIGN KEY (project_id, provider)
    REFERENCES providers(project_id, name)
    ON DELETE CASCADE;

ALTER TABLE rule_type
    DROP CONSTRAINT rule_type_project_id_fkey;

ALTER TABLE rule_type
    ADD CONSTRAINT rule_type_project_id_provider_fkey
    FOREIGN KEY (project_id, provider)
    REFERENCES providers(project_id, name)
    ON DELETE CASCADE;

ALTER TABLE profiles
    DROP CONSTRAINT profiles_project_id_fkey;

ALTER TABLE profiles
    ADD CONSTRAINT profiles_project_id_provider_fkey
    FOREIGN KEY (project_id, provider)
    REFERENCES providers(project_id, name)
    ON DELETE CASCADE;

