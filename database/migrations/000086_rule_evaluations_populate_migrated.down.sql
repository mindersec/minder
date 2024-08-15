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

-- essentially replays migration #83
ALTER TABLE rule_evaluations ADD COLUMN rule_entity_id UUID REFERENCES evaluation_rule_entities(id) ON DELETE CASCADE;

UPDATE rule_evaluations
SET rule_entity_id = ere.id
FROM evaluation_rule_entities AS ere
WHERE rule_evaluations.rule_entity_id IS NULL
  AND rule_evaluations.entity = 'artifact'::entities
  AND rule_evaluations.rule_instance_id = ere.rule_id
  AND rule_evaluations.artifact_id = ere.artifact_id;

UPDATE rule_evaluations
SET rule_entity_id = ere.id
FROM evaluation_rule_entities AS ere
WHERE rule_evaluations.rule_entity_id IS NULL
  AND rule_evaluations.entity = 'pull_request'::entities
  AND rule_evaluations.rule_instance_id = ere.rule_id
  AND rule_evaluations.pull_request_id = ere.pull_request_id;

UPDATE rule_evaluations
SET rule_entity_id = ere.id
FROM evaluation_rule_entities AS ere
WHERE rule_evaluations.rule_entity_id IS NULL
  AND rule_evaluations.entity = 'repository'::entities
  AND rule_evaluations.rule_instance_id = ere.rule_id
  AND rule_evaluations.repository_id = ere.repository_id;

COMMIT;