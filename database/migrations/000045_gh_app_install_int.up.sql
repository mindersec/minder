-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- migrate app_installation_id to int. This presumes that the column only contains integers
ALTER TABLE provider_github_app_installations
    ALTER COLUMN app_installation_id TYPE BIGINT USING app_installation_id::BIGINT;

COMMIT;
