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

-- Add checkpoint column to the evaluation_statuses table.
-- Note that the default value is an empty JSON object. This is fine for
-- now because the checkpoint is not used in the application yet.
-- There will be a separate migration to populate the checkpoint column.
ALTER TABLE evaluation_statuses ADD COLUMN checkpoint JSONB DEFAULT '{}' NOT NULL;

-- Add an index on the checkpoint column for faster lookups.
CREATE INDEX evaluation_statuses_checkpoint_idx ON evaluation_statuses USING GIN (checkpoint);

COMMIT;