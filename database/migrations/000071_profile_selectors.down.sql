-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- drop the index on the project column
DROP INDEX IF EXISTS idx_profile_selectors_on_profile;

-- drop the profile_selectors table
DROP TABLE IF EXISTS profile_selectors;

COMMIT;
