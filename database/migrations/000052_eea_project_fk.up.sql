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

-- Delete eea locks and flush caches associated with a project that gets deleted
-- The FKs to reset are fk_entity_execution_lock_project_id and fk_flush_cache_project_id
ALTER TABLE entity_execution_lock DROP CONSTRAINT fk_entity_execution_lock_project_id;
ALTER TABLE entity_execution_lock ADD CONSTRAINT fk_entity_execution_lock_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE flush_cache DROP CONSTRAINT fk_flush_cache_project_id;
ALTER TABLE flush_cache ADD CONSTRAINT fk_flush_cache_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

COMMIT;