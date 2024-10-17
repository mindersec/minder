-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Add a lower-case unique index for rule name, this mirrors the changes in migration #31
CREATE UNIQUE INDEX rule_instances_lower_name ON rule_instances (profile_id, entity_type, lower(name));
ALTER TABLE rule_instances DROP CONSTRAINT IF EXISTS rule_instances_profile_id_entity_type_name_key;

-- Add "migrated" column to "rule_evaluations". This replaces the "rule_entity_id"
-- column - using a boolean is simpler, and avoids making changes to DB tests.
ALTER TABLE rule_evaluations
ADD COLUMN migrated BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;
