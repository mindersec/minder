-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE entity_execution_lock DROP COLUMN IF EXISTS repository_id;
ALTER TABLE entity_execution_lock DROP COLUMN IF EXISTS artifact_id;
ALTER TABLE entity_execution_lock DROP COLUMN IF EXISTS pull_request_id;

ALTER TABLE flush_cache DROP COLUMN IF EXISTS repository_id;
ALTER TABLE flush_cache DROP COLUMN IF EXISTS artifact_id;
ALTER TABLE flush_cache DROP COLUMN IF EXISTS pull_request_id;

-- make project_id not nullable
ALTER TABLE entity_execution_lock ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE flush_cache ALTER COLUMN project_id SET NOT NULL;

COMMIT;
