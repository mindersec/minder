-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- repositories.repo_id was stored as an int32, but it is an int64 from GitHub
-- Note that this takes an exclusive write lock on the table, which should be okay
-- as long as the actual number of repositories is small.
ALTER TABLE repositories ALTER COLUMN repo_id TYPE BIGINT;
