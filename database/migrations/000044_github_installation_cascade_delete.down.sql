-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE provider_github_app_installations
    DROP CONSTRAINT provider_github_app_installations_project_id_fkey;

ALTER TABLE provider_github_app_installations
    ADD CONSTRAINT provider_github_app_installations_project_id_fkey
        FOREIGN KEY (project_id)
            REFERENCES projects(id);

COMMIT;
