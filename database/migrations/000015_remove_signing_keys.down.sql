-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- signing_keys table
CREATE TABLE signing_keys (
                              id SERIAL PRIMARY KEY,
                              project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
                              private_key TEXT NOT NULL,
                              public_key TEXT NOT NULL,
                              passphrase TEXT NOT NULL,
                              key_identifier TEXT NOT NULL,
                              created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                              updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
