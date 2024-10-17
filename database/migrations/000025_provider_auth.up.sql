-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

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
