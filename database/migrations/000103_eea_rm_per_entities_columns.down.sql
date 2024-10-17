-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE entity_execution_lock ADD COLUMN IF NOT EXISTS repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE;
ALTER TABLE entity_execution_lock ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE;
ALTER TABLE entity_execution_lock ADD COLUMN IF NOT EXISTS pull_request_id UUID REFERENCES pull_requests(id) ON DELETE CASCADE;

ALTER TABLE flush_cache ADD COLUMN IF NOT EXISTS repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE;
ALTER TABLE flush_cache ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE;
ALTER TABLE flush_cache ADD COLUMN IF NOT EXISTS pull_request_id UUID REFERENCES pull_requests(id) ON DELETE CASCADE;

-- make project_id nullable
ALTER TABLE entity_execution_lock ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE flush_cache ALTER COLUMN project_id DROP NOT NULL;

COMMIT;
