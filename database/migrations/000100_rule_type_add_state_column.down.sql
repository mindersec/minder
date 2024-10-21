-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE rule_type DROP COLUMN release_phase;
DROP TYPE release_status;

COMMIT;
