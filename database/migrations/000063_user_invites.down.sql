-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- drop the index on the project column
DROP INDEX IF EXISTS idx_user_invites_project;
-- drop the index on the email column
DROP INDEX IF EXISTS idx_user_invites_email;

-- drop the user_invites table
DROP TABLE IF EXISTS user_invites;

COMMIT;
