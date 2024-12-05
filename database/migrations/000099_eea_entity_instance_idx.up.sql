-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Add entity_instance_id as a unique index to the entity_execution_lock and flush_cache tables

CREATE UNIQUE INDEX entity_execution_lock_entity_instance_idx ON entity_execution_lock (entity_instance_id);
CREATE UNIQUE INDEX flush_cache_entity_instance_idx ON flush_cache (entity_instance_id);

COMMIT;
