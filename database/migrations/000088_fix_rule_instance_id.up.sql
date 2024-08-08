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

UPDATE rule_evaluations
SET rule_instance_id = ri.id
FROM rule_instances AS ri
WHERE rule_evaluations.profile_id = ri.profile_id
    AND rule_evaluations.rule_name = ri.name
    AND rule_evaluations.rule_type_id = ri.rule_type_id
    AND rule_evaluations.entity = ri.entity_type;

-- the previous migration may have flagged some rows as migrated which should
-- not have been migrated. Redo them.

UPDATE rule_evaluations AS re
SET migrated = TRUE
FROM evaluation_rule_entities AS ere
WHERE re.entity = 'pull_request'::entities
  AND ere.entity_type = 'pull_request'::entities
  AND ere.pull_request_id = re.pull_request_id
  AND ere.rule_id = re.rule_instance_id;

UPDATE rule_evaluations AS re
SET migrated = TRUE
FROM evaluation_rule_entities AS ere
WHERE re.entity = 'artifact'::entities
  AND ere.entity_type = 'artifact'::entities
  AND ere.artifact_id = re.artifact_id
  AND ere.rule_id = re.rule_instance_id;

UPDATE rule_evaluations AS re
SET migrated = TRUE
FROM evaluation_rule_entities AS ere
WHERE re.entity = 'repository'::entities
  AND ere.entity_type = 'repository'::entities
  AND ere.repository_id = re.repository_id
  AND ere.rule_id = re.rule_instance_id;

-- ensure that any non-migrated rows are set to false

UPDATE rule_evaluations
SET migrated = FALSE
WHERE entity = 'artifact'::entities
AND id IN (
    SELECT re.id FROM rule_evaluations AS re
    LEFT JOIN evaluation_rule_entities AS ere
        ON  re.artifact_id = ere.artifact_id
        AND re.rule_instance_id = ere.rule_id
        AND re.entity = 'artifact'::entities
        AND ere.entity_type = 'artifact'::entities
    WHERE ere.id IS NULL
);

UPDATE rule_evaluations
SET migrated = FALSE
WHERE entity = 'pull_request'::entities
AND id IN (
    SELECT re.id FROM rule_evaluations AS re
    LEFT JOIN evaluation_rule_entities AS ere
        ON  re.pull_request_id = ere.pull_request_id
        AND re.rule_instance_id = ere.rule_id
        AND re.entity = 'pull_request'::entities
        AND ere.entity_type = 'pull_request'::entities
    WHERE ere.id IS NULL
);

UPDATE rule_evaluations
SET migrated = FALSE
WHERE entity = 'repository'::entities
AND id IN (
    SELECT re.id FROM rule_evaluations AS re
    LEFT JOIN evaluation_rule_entities AS ere
        ON  re.repository_id = ere.repository_id
        AND re.rule_instance_id = ere.rule_id
        AND re.entity = 'repository'::entities
        AND ere.entity_type = 'repository'::entities
    WHERE ere.id IS NULL
);

COMMIT;