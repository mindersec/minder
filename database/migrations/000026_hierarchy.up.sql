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

ALTER TABLE repositories
  ADD COLUMN provider_id UUID REFERENCES providers(id) ON DELETE CASCADE;

-- set the provider_id for all existing repositories
UPDATE repositories
  SET provider_id = providers.id
  FROM providers
  WHERE repositories.provider = providers.name AND repositories.project_id = providers.project_id;

-- Make provider_id not nullable
ALTER TABLE repositories
  ALTER COLUMN provider_id SET NOT NULL;

-- change the project_id column from being a foreign key from the providers
-- table to being a foreign key from the projects table
ALTER TABLE repositories
  DROP CONSTRAINT repositories_project_id_provider_fkey;

-- Add constraints so project_id is a foreign key to the projects table
ALTER TABLE repositories
  ADD CONSTRAINT repositories_project_id_fkey
  FOREIGN KEY (project_id)
  REFERENCES projects(id)
  ON DELETE CASCADE;