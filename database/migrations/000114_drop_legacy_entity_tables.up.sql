-- SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Drop legacy entity tables in correct order (respecting foreign keys)
-- These tables have been fully replaced by the unified entity_instances and properties tables.
-- All code has been migrated to use the new entity model.

-- WARNING: This is a destructive operation. Ensure backups exist before running.

DROP TABLE IF EXISTS pull_requests CASCADE;
DROP TABLE IF EXISTS artifacts CASCADE;
DROP TABLE IF EXISTS repositories CASCADE;

COMMIT;
