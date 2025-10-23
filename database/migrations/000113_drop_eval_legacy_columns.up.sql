-- SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Remove legacy entity ID columns from evaluation_rule_entities table.
-- These columns have been replaced by the unified entity_instance_id column.
-- Migration 097 already made entity_instance_id NOT NULL, ensuring all rows
-- have been migrated to use the new unified entity model.

ALTER TABLE evaluation_rule_entities
  DROP COLUMN IF EXISTS repository_id,
  DROP COLUMN IF EXISTS pull_request_id,
  DROP COLUMN IF EXISTS artifact_id;

COMMIT;
