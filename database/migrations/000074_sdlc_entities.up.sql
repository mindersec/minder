-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN TRANSACTION;

ALTER TYPE entities ADD VALUE 'release';
ALTER TYPE entities ADD VALUE 'pipeline_run';
ALTER TYPE entities ADD VALUE 'task_run';
ALTER TYPE entities ADD VALUE 'build';

COMMIT;
