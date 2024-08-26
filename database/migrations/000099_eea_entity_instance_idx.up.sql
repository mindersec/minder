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

-- Add entity_instance_id as a unique index to the entity_execution_lock and flush_cache tables

CREATE UNIQUE INDEX entity_execution_lock_entity_instance_idx ON entity_execution_lock (entity_instance_id);
CREATE UNIQUE INDEX flush_cache_entity_instance_idx ON flush_cache (entity_instance_id);

COMMIT;