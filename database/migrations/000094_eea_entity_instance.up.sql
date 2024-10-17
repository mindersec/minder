-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0


BEGIN;

-- Add optional FK towards entity_instances(id) in entity_execution_lock and flush_cache

ALTER TABLE entity_execution_lock ADD COLUMN entity_instance_id UUID;
ALTER TABLE entity_execution_lock ADD CONSTRAINT fk_entity_instance_id FOREIGN KEY (entity_instance_id) REFERENCES entity_instances(id) ON DELETE CASCADE;

ALTER TABLE flush_cache ADD COLUMN entity_instance_id UUID;
ALTER TABLE flush_cache ADD CONSTRAINT fk_entity_instance_id FOREIGN KEY (entity_instance_id) REFERENCES entity_instances(id) ON DELETE CASCADE;

COMMIT;
