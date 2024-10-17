-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Delete eea locks and flush caches associated with a project that gets deleted
-- The FKs to reset are fk_entity_execution_lock_project_id and fk_flush_cache_project_id
ALTER TABLE entity_execution_lock DROP CONSTRAINT fk_entity_execution_lock_project_id;
ALTER TABLE entity_execution_lock ADD CONSTRAINT fk_entity_execution_lock_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE flush_cache DROP CONSTRAINT fk_flush_cache_project_id;
ALTER TABLE flush_cache ADD CONSTRAINT fk_flush_cache_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

COMMIT;
