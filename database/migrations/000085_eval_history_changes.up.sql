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

-- Add a lower-case unique index for rule name, this mirrors the changes in migration #31
CREATE UNIQUE INDEX rule_instances_lower_name ON rule_instances (profile_id, entity_type, lower(name));
ALTER TABLE rule_instances DROP CONSTRAINT IF EXISTS rule_instances_profile_id_entity_type_name_key;

-- Add "migrated" column to "rule_evaluations". This replaces the "rule_entity_id"
-- column - using a boolean is simpler, and avoids making changes to DB tests.
ALTER TABLE rule_evaluations
ADD COLUMN migrated BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;