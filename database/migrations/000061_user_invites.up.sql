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

-- describes the invite status
CREATE TYPE invite_status AS ENUM ('pending', 'accepted', 'declined', 'expired', 'revoked');
-- reflects the roles we have in OpenFGA
CREATE TYPE user_role AS ENUM ('admin', 'editor', 'viewer', 'policy_writer', 'permissions_manager');

CREATE TABLE IF NOT EXISTS user_invites(
    id           UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    email        TEXT NOT NULL,
    status       invite_status NOT NULL DEFAULT 'pending',
    role         user_role NOT NULL DEFAULT 'viewer',
    project      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    invitee      INTEGER NULL REFERENCES users(id) ON DELETE CASCADE,
    sponsor      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code         TEXT NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    -- ensure there's one invite for this email per project
    UNIQUE (email, project),
    -- ensure code/nonce is unique
    UNIQUE (code)
);

-- create an index on the email column
CREATE INDEX idx_user_invites_email ON user_invites(email);

-- create an index on the project column
CREATE INDEX idx_user_invites_project ON user_invites(project);

-- create an index on the invitee column
CREATE INDEX idx_user_invites_invitee ON user_invites(invitee);

-- create an index on the invited_by column
CREATE INDEX idx_user_invites_sponsor ON user_invites(sponsor);

COMMIT;
