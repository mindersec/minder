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

UPDATE rule_evaluations AS re
SET migrated = TRUE
FROM evaluation_rule_entities AS ere
WHERE re.entity = 'pull_request'::entities
  AND ere.entity_type = 'pull_request'::entities
  AND ere.pull_request_id = re.pull_request_id
  AND re.migrated = FALSE;

UPDATE rule_evaluations AS re
SET migrated = TRUE
FROM evaluation_rule_entities AS ere
WHERE re.entity = 'artifact'::entities
  AND ere.entity_type = 'artifact'::entities
  AND ere.artifact_id = re.artifact_id
  AND re.migrated = FALSE;

UPDATE rule_evaluations AS re
SET migrated = TRUE
FROM evaluation_rule_entities AS ere
WHERE re.entity = 'repository'::entities
  AND ere.entity_type = 'repository'::entities
  AND ere.repository_id = re.repository_id
  AND re.migrated = FALSE;

COMMIT;