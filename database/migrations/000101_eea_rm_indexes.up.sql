-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

DROP INDEX IF EXISTS entity_execution_lock_idx;
DROP INDEX IF EXISTS flush_cache_idx;

COMMIT;
