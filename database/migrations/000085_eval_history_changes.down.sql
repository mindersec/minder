-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE rule_instances
ADD CONSTRAINT rule_instances_profile_id_entity_type_name_key UNIQUE (profile_id, entity_type, name);
DROP INDEX rule_instances_lower_name;

ALTER TABLE rule_evaluations DROP COLUMN migrated;

COMMIT;
