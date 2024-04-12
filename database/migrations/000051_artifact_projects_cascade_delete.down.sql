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

-- Artifact changes

BEGIN;

-- delete artifacts that are associated with a project that gets deleted
-- otherwise removing a project would cause:
-- "update or delete on table "projects" violates foreign key constraint "fk_artifacts_project_id" on table "artifacts""
ALTER TABLE artifacts DROP CONSTRAINT fk_artifacts_project_id;
ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_project_id FOREIGN KEY (project_id) REFERENCES projects (id);

-- delete artifacts that are associated with a provider that gets deleted
-- same reason as above. This is mostly a noop now because we also have a on delete cascade on the repository_id column
-- which while technically nullable is always set to a value, but would bite us in the future
ALTER TABLE artifacts DROP CONSTRAINT fk_artifacts_provider_id_and_name;
ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_provider_id_and_name FOREIGN KEY (provider_id, provider_name) REFERENCES providers (id, name);

COMMIT;