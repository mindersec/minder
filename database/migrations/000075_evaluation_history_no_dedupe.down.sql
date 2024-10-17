-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE evaluation_statuses ADD COLUMN evaluation_times TIMESTAMPZ[] NOT NULL DEFAULT ARRAY[NOW()]::TIMESTAMP[];
ALTER TABLE evaluation_statuses RENAME COLUMN evaluation_time TO most_recent_evaluation;

COMMIT;
