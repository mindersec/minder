-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE evaluation_statuses DROP COLUMN evaluation_times;
ALTER TABLE evaluation_statuses RENAME COLUMN most_recent_evaluation TO evaluation_time;

COMMIT;
