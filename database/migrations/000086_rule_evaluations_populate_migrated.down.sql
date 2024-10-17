-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

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
