-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Drop the 'is_protected' column
ALTER TABLE roles
    DROP COLUMN IF EXISTS is_protected;

-- transaction commit
COMMIT;
