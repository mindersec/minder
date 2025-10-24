-- SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Restore legacy entity ID columns to evaluation_rule_entities table.
-- This rollback recreates the columns and attempts to restore data from entity_instances.

ALTER TABLE evaluation_rule_entities
  ADD COLUMN IF NOT EXISTS repository_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS pull_request_id UUID REFERENCES pull_requests(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE;

-- Restore data by joining through entity_instances to legacy tables.
-- The entity_instance_id in evaluation_rule_entities has the same UUID as the legacy table IDs.

-- Restore repository_id for repository entities
UPDATE evaluation_rule_entities AS ere
SET repository_id = ei.id
FROM entity_instances AS ei
WHERE ere.entity_instance_id = ei.id
  AND ei.entity_type = 'repository'::entities
  AND EXISTS (SELECT 1 FROM repositories WHERE id = ei.id);

-- Restore artifact_id for artifact entities
UPDATE evaluation_rule_entities AS ere
SET artifact_id = ei.id
FROM entity_instances AS ei
WHERE ere.entity_instance_id = ei.id
  AND ei.entity_type = 'artifact'::entities
  AND EXISTS (SELECT 1 FROM artifacts WHERE id = ei.id);

-- Restore pull_request_id for pull_request entities
UPDATE evaluation_rule_entities AS ere
SET pull_request_id = ei.id
FROM entity_instances AS ei
WHERE ere.entity_instance_id = ei.id
  AND ei.entity_type = 'pull_request'::entities
  AND EXISTS (SELECT 1 FROM pull_requests WHERE id = ei.id);

COMMIT;
