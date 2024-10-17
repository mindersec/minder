-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Drop the checkpoint column from the evaluation_statuses table.
ALTER TABLE evaluation_statuses DROP COLUMN checkpoint;

COMMIT;
