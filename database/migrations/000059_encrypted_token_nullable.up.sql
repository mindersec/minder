-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE provider_access_tokens ALTER COLUMN encrypted_token DROP NOT NULL;

COMMIT;
