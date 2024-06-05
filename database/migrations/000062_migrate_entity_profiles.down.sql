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

-- in case we need rollback, wipe out the rule_instances table and mark all rows
-- in entity_profiles as unmigrated. A re-run of the migration will recreate all
-- rows in the rule_instances of table since we are dual writing at this point.
DELETE FROM rule_instances;
UPDATE entity_profiles SET migrated = FALSE;

COMMIT;