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

CREATE TABLE IF NOT EXISTS user_invites (
    code          TEXT NOT NULL PRIMARY KEY,
    email         TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'viewer',
    project       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    sponsor       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

-- create an index on the project column
CREATE INDEX IF NOT EXISTS idx_user_invites_project ON user_invites(project);
-- create an index on the email column
CREATE INDEX IF NOT EXISTS idx_user_invites_email ON user_invites(email);

COMMIT;
