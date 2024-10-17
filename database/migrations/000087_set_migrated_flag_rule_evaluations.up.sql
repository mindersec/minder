-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

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
