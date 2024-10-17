-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

CREATE TABLE IF NOT EXISTS user_invites (
    code          TEXT NOT NULL PRIMARY KEY,
    email         TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'viewer',
    project       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    sponsor       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

-- create an index on the project column
CREATE INDEX IF NOT EXISTS idx_user_invites_project ON user_invites(project);
-- create an index on the email column
CREATE INDEX IF NOT EXISTS idx_user_invites_email ON user_invites(email);

COMMIT;
