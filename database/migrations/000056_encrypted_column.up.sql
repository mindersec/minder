-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- add columns for new encrypted data format

ALTER TABLE provider_access_tokens ADD COLUMN encrypted_access_token JSONB;
ALTER TABLE session_store ADD COLUMN encrypted_redirect TEXT;
