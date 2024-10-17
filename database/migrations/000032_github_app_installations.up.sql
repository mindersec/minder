-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- providers becomes a star schema, where the provider is the central table,
-- and different classes of providers have different provider_* tables that
-- join with the providers table to get the credential parameters.

CREATE TABLE provider_github_app_installations (
    app_installation_id TEXT PRIMARY KEY,
    provider_id         UUID,
    organization_id     BIGINT    NOT NULL,
    -- If provider_id is NULL, this records the GitHub UserID (numeric) of the
    -- user who completed the app installation flow for later connection with
    -- a project.
    enrolling_user_id   TEXT,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE NULLS DISTINCT (provider_id), -- NULL provider_ids are unclaimed.
    UNIQUE (organization_id),            -- add an index on organization_id
    FOREIGN KEY (provider_id) REFERENCES providers (id) ON DELETE CASCADE
);
