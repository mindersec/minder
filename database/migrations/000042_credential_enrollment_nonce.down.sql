-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE provider_github_app_installations
    DROP CONSTRAINT provider_github_app_installations_project_id_fkey;

ALTER TABLE provider_github_app_installations DROP COLUMN project_id;

ALTER TABLE provider_github_app_installations DROP COLUMN enrollment_nonce;

ALTER TABLE provider_access_tokens DROP COLUMN enrollment_nonce;

COMMIT;
