-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

DROP INDEX IF EXISTS flush_cache_idx;
DROP INDEX IF EXISTS entity_execution_lock_idx;

DROP TABLE IF EXISTS flush_cache;
DROP TABLE IF EXISTS entity_execution_lock;
