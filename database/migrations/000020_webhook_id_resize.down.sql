-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- repositories.webhook_id was stored as an int32, but it is an int64 from GitHub
-- Note that this takes an exclusive write lock on the table, which should be okay
-- as long as the actual number of repositories is small.
-- Given that we're shrinking the range, we'll need to check that the values are
-- all within range; if not, this migration will fail and need manual intervention.

BEGIN;
-- webhook_id column is NOT NULL, so this will *fail* and abort transaction.
UPDATE repositories SET webhook_id = NULL WHERE webhook_id > 2147483647;
ALTER TABLE repositories ALTER COLUMN webhook_id TYPE INTEGER;
COMMIT;
