-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

DROP INDEX IF EXISTS idx_evaluation_outputs_evaluation_id;
DROP TABLE IF EXISTS evaluation_outputs;

COMMIT;
