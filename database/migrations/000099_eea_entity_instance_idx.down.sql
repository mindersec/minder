-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Drop the unique index on entity_instance_id from the entity_execution_lock and flush_cache tables

DROP INDEX entity_execution_lock_entity_instance_idx ON entity_execution_lock;
DROP INDEX flush_cache_entity_instance_idx ON flush_cache;

COMMIT;
