-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Postgres only needs to rewrite metadata (and not the column data) when altering
-- a column from timezoneless when timezone='UTC' is set in the session.
-- Ref: https://www.postgresql.org/docs/release/12.0/

SET timezone = 'UTC';
ALTER TABLE properties
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ,
  ALTER updated_at SET DEFAULT NOW()::timestamptz;  -- Default needs to be set explicitly after type change

ALTER TABLE entity_instances
    ALTER created_at TYPE TIMESTAMPTZ,
    ALTER created_at SET DEFAULT NOW()::timestamptz;

-- There are more TIMESTAMP columns, but these are affecting unit tests in some timezones.

COMMIT;
