-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE provider_github_app_installations ADD COLUMN enrollment_nonce TEXT;

ALTER TABLE provider_github_app_installations ADD COLUMN project_id UUID;

ALTER TABLE provider_github_app_installations
    ADD CONSTRAINT provider_github_app_installations_project_id_fkey
        FOREIGN KEY (project_id)
            REFERENCES projects(id);

ALTER TABLE provider_access_tokens ADD COLUMN enrollment_nonce TEXT;

COMMIT;
