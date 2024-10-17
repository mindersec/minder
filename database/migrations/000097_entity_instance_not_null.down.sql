-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Make entity_instance_id nullable in entity_execution_lock, flush_cache and evaluation_rule_entities.

ALTER TABLE entity_execution_lock ALTER COLUMN entity_instance_id DROP NOT NULL;
ALTER TABLE flush_cache ALTER COLUMN entity_instance_id DROP NOT NULL;
ALTER TABLE evaluation_rule_entities ALTER COLUMN entity_instance_id DROP NOT NULL;

COMMIT;
