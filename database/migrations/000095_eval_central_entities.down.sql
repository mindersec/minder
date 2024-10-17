-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Drop optional FK towards entity_instances(id) in evaluation_rule_entities

ALTER TABLE evaluation_rule_entities DROP COLUMN entity_instance_id;

COMMIT;
