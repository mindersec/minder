-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- backfill rows which don't have an rule instance ID
UPDATE rule_evaluations
SET rule_instance_id = ri.id
FROM rule_instances AS ri
WHERE rule_evaluations.rule_instance_id IS NULL
    AND rule_evaluations.rule_name = ri.name
    AND rule_evaluations.rule_type_id = ri.rule_type_id;

-- backfill rows which don't have an rule entity ID
-- This process only matches up rule_evaluations which have a corresponding
-- evaluation_rule_entities entry. Entities which were last evaluated before
-- the introduction of the evaluation history tables will not have entries
-- in evaluation_rule_entities - this will be addressed in a future PR.

-- In principle, we could write a single query which matches the three types of
-- entity we care about. Unfortunately, the rule_evaluations table may contain
-- the repository ID for non-repo entities. In order to work around this, backfill
-- each type of entity separately.
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

-- make field mandatory
ALTER TABLE rule_evaluations ALTER COLUMN rule_instance_id SET NOT NULL;
-- note that rule_entity_id is still expected to contain nulls until we backfill
-- evaluation state which predates the evaluation history tables.

COMMIT;
