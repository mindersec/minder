-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE provider_github_app_installations ADD COLUMN is_org BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;
