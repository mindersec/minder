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

CREATE TYPE authorization_flow AS ENUM ('user_input', 'oauth2_authorization_code_flow', 'github_app_flow', 'none');

ALTER TABLE providers ADD COLUMN auth_flows authorization_flow ARRAY NOT NULL DEFAULT '{}';

-- Iterate providers that implement github and add the `github_app_flow` and `user_input` to their auth_flows
DO $$
DECLARE
  provider_name TEXT;
BEGIN
    FOR provider_name IN SELECT name FROM providers WHERE name = 'github' LOOP
        UPDATE providers SET auth_flows = ARRAY['github_app_flow'::authorization_flow, 'user_input'::authorization_flow] WHERE name = provider_name;
    END LOOP;
END $$;