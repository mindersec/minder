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

-- drop the indexes
DROP INDEX IF EXISTS idx_user_invites_email;
DROP INDEX IF EXISTS idx_user_invites_project;
DROP INDEX IF EXISTS idx_user_invites_invitee;
DROP INDEX IF EXISTS idx_user_invites_sponsor;

-- drop the table
DROP TABLE IF EXISTS user_invites;

-- drop the types
DROP TYPE IF EXISTS invite_status;
DROP TYPE IF EXISTS user_role;

COMMIT;
