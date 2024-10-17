-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE entity_execution_lock DROP COLUMN entity_instance_id;
ALTER TABLE flush_cache DROP COLUMN entity_instance_id;

COMMIT;
