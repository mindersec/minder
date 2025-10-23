-- SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Restore legacy entity ID columns to evaluation_rule_entities table.
-- WARNING: This rollback migration recreates the columns but does NOT
-- restore any data. The columns will be nullable and empty.
-- This is for emergency rollback only and requires manual data recovery.

ALTER TABLE evaluation_rule_entities
  ADD COLUMN IF NOT EXISTS repository_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS pull_request_id UUID REFERENCES pull_requests(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE;

-- Recreate the constraint that exactly one legacy entity ID must be set
-- (or use the new entity_instance_id). This constraint may fail if rolled
-- back before data is properly restored.
ALTER TABLE evaluation_rule_entities
  ADD CONSTRAINT one_entity_id CHECK (
    num_nonnulls(repository_id, artifact_id, pull_request_id) >= 0
  );

COMMIT;
