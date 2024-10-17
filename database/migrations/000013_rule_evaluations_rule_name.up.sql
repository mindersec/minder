-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Begin transaction
BEGIN;

-- Add rule_name to rule_evaluations
ALTER TABLE rule_evaluations
  ADD COLUMN rule_name TEXT;

-- Drop the existing unique index on rule_evaluations
DROP INDEX IF EXISTS rule_evaluations_results_idx;

-- Recreate the unique index with rule_name
CREATE UNIQUE INDEX rule_evaluations_results_idx
  ON rule_evaluations (profile_id, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
                       entity, rule_type_id, COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID),
                       rule_name) NULLS NOT DISTINCT;

-- transaction commit
COMMIT;
