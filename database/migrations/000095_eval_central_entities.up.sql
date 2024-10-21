-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Add optional FK towards entity_instances(id) in evaluation_rule_entities

ALTER TABLE evaluation_rule_entities ADD COLUMN entity_instance_id UUID;
ALTER TABLE evaluation_rule_entities ADD CONSTRAINT fk_entity_instance_id FOREIGN KEY (entity_instance_id) REFERENCES entity_instances(id) ON DELETE CASCADE;

COMMIT;
