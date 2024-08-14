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

-- Drop the foreign key constraint and then recreate it with the ON DELETE CASCADE option
BEGIN;

ALTER TABLE entity_instances DROP CONSTRAINT entity_instances_project_id_fkey;

ALTER TABLE entity_instances ADD CONSTRAINT entity_instances_project_id_fkey FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

-- Do the same for the provider ID, since deleting a provider should delete all entities associated with it
ALTER TABLE entity_instances DROP CONSTRAINT entity_instances_provider_id_fkey;

ALTER TABLE entity_instances ADD CONSTRAINT entity_instances_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE;

COMMIT;