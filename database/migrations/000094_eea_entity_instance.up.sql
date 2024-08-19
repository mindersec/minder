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

-- Add optional FK towards entity_instances(id) in entity_execution_lock and flush_cache

ALTER TABLE entity_execution_lock ADD COLUMN entity_instance_id UUID;
ALTER TABLE entity_execution_lock ADD CONSTRAINT fk_entity_instance_id FOREIGN KEY (entity_instance_id) REFERENCES entity_instances(id) ON DELETE CASCADE;

ALTER TABLE flush_cache ADD COLUMN entity_instance_id UUID;
ALTER TABLE flush_cache ADD CONSTRAINT fk_entity_instance_id FOREIGN KEY (entity_instance_id) REFERENCES entity_instances(id) ON DELETE CASCADE;

COMMIT;