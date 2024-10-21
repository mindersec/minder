-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Make entity_instance_id not nullable in entity_execution_lock, flush_cache and evaluation_rule_entities.
-- Note that at this point it is verified that the entity_instance_id is not null in all these tables.

ALTER TABLE entity_execution_lock ALTER COLUMN entity_instance_id SET NOT NULL;
ALTER TABLE flush_cache ALTER COLUMN entity_instance_id SET NOT NULL;
ALTER TABLE evaluation_rule_entities ALTER COLUMN entity_instance_id SET NOT NULL;

COMMIT;
