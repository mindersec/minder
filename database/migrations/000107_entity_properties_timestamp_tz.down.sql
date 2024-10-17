-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

SET timezone = 'UTC';
ALTER TABLE properties
  ALTER COLUMN updated_at TYPE TIMESTAMP,
  ALTER updated_at SET DEFAULT NOW();  -- Default needs to be set explicitly after type change

ALTER TABLE entity_instances
    ALTER created_at TYPE TIMESTAMP,
    ALTER created_at SET DEFAULT NOW();

-- There are more TIMESTAMP columns, but these are affecting unit tests in some timezones.

COMMIT;
