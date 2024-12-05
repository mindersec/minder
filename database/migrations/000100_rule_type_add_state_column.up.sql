-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

CREATE TYPE release_status AS ENUM ('alpha', 'beta', 'ga', 'deprecated');
ALTER TABLE rule_type ADD COLUMN release_phase release_status;
UPDATE rule_type SET release_phase = 'alpha' WHERE release_phase IS NULL;
ALTER TABLE rule_type
  ALTER COLUMN release_phase SET DEFAULT 'ga',
  ALTER COLUMN release_phase SET NOT NULL;

COMMIT;
