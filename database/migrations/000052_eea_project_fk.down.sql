-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE entity_execution_locks DROP CONSTRAINT fk_entity_execution_lock_project_id;
ALTER TABLE entity_execution_locks ADD CONSTRAINT fk_entity_execution_lock_project_id FOREIGN KEY (project_id) REFERENCES projects (id);

ALTER TABLE flush_caches DROP CONSTRAINT fk_flush_cache_project_id;
ALTER TABLE flush_caches ADD CONSTRAINT fk_flush_cache_project_id FOREIGN KEY (project_id) REFERENCES projects (id);

COMMIT;
