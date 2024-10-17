-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE provider_access_tokens DROP COLUMN encrypted_access_token;
ALTER TABLE session_store DROP COLUMN encrypted_redirect;
