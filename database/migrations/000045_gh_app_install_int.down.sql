-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE provider_github_app_installations
    ALTER COLUMN app_installation_id TYPE TEXT USING app_installation_id::TEXT;

COMMIT;
