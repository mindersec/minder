-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- This replaces the rule_entity_id column with the migrated column
UPDATE rule_evaluations
SET migrated = TRUE
WHERE rule_entity_id IS NOT NULL;

ALTER TABLE rule_evaluations DROP COLUMN rule_entity_id;

COMMIT;
