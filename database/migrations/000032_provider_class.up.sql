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

CREATE TYPE provider_class AS enum ('github', 'gh-app');

ALTER TABLE providers
  -- Fills existing rows with 'github'
  ADD COLUMN class provider_class DEFAULT 'github' NOT NULL;

-- providers becomes a star schema, where the provider is the central table,
-- and different classes of providers have different provider_* tables that
-- join with the providers table to get the credential parameters.

CREATE TABLE provider_app_installations (
    app_installation_id TEXT PRIMARY KEY,
    provider_id UUID,
    organization_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE NULLS DISTINCT (provider_id),  -- NULL provider_ids are unclaimed.
    UNIQUE (organization_id),  -- add an index on organization_id
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

