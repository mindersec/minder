-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Add checkpoint column to the evaluation_statuses table.
-- Note that the default value is an empty JSON object. This is fine for
-- now because the checkpoint is not used in the application yet.
-- There will be a separate migration to populate the checkpoint column.
ALTER TABLE evaluation_statuses ADD COLUMN checkpoint JSONB DEFAULT '{}' NOT NULL;

-- Add an index on the checkpoint column for faster lookups.
CREATE INDEX evaluation_statuses_checkpoint_idx ON evaluation_statuses USING GIN (checkpoint);

COMMIT;
