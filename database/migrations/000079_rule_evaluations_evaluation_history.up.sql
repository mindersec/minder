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

-- link each entry in the rule_evaluations table to the evaluation_rule_entities
-- table, and to the rule_instances table. This will simplify migrating statuses
-- from the old status tables over to the new history tables.
ALTER TABLE rule_evaluations ADD COLUMN rule_entity_id UUID REFERENCES evaluation_rule_entities(id) ON DELETE CASCADE;
ALTER TABLE rule_evaluations ADD COLUMN rule_instance_id UUID REFERENCES rule_instances(id) ON DELETE CASCADE;

-- Fix some omissions from the previous PR

-- First, add an ON DELETE CASCADE to the profile_id FK
ALTER TABLE latest_evaluation_statuses DROP CONSTRAINT latest_evaluation_statuses_profile_id_fkey;
ALTER TABLE latest_evaluation_statuses
    ADD CONSTRAINT latest_evaluation_statuses_profile_id_fkey
    FOREIGN KEY (profile_id)
    REFERENCES profiles(id)
    ON DELETE CASCADE;

-- recreate index with default name
DROP INDEX IF EXISTS idx_profile_id;
CREATE INDEX ON latest_evaluation_statuses(profile_id);

COMMIT;