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

-- backfill rows which don't have a profile ID
UPDATE latest_evaluation_statuses
SET profile_id = ri.profile_id
FROM rule_instances AS ri
JOIN evaluation_rule_entities AS ere ON ere.rule_id = ri.id
JOIN latest_evaluation_statuses AS les ON les.rule_entity_id = ere.id
WHERE les.profile_id IS NULL;

-- make field mandatory
ALTER TABLE latest_evaluation_statuses ALTER COLUMN profile_id SET NOT NULL;

COMMIT;