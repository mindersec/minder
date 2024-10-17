-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;


ALTER TABLE evaluation_statuses ALTER COLUMN most_recent_evaluation TYPE TIMESTAMP;
ALTER TABLE evaluation_statuses ALTER COLUMN most_recent_evaluation SET DEFAULT NOW();
ALTER TABLE evaluation_statuses ALTER COLUMN evaluation_times TYPE TIMESTAMP[];
ALTER TABLE evaluation_statuses ALTER COLUMN evaluation_times SET DEFAULT ARRAY[NOW()]::TIMESTAMP[];

COMMIT;
