-- Copyright 2024 Stacklok, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

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