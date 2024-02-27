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

-- repositories.webhook_id was stored as an int32, but it is an int64 from GitHub
-- Note that this takes an exclusive write lock on the table, which should be okay
-- as long as the actual number of repositories is small.
-- Given that we're shrinking the range, we'll need to check that the values are
-- all within range; if not, this migration will fail and need manual intervention.

BEGIN;
-- repo_id column is NOT NULL, so this will *fail* and abort transaction.
UPDATE repositories SET webhook_id = NULL WHERE webhook_id > 2147483647;
ALTER TABLE repositories ALTER COLUMN webhook_id TYPE INTEGER;
COMMIT;
