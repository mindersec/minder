-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

CREATE INDEX IF NOT EXISTS entity_instances_originated_from_idx
    ON entity_instances (originated_from)
    WHERE originated_from IS NOT NULL;

COMMIT;
