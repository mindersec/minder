-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE migration_profile_backfill_log (
    profile_id UUID PRIMARY KEY,
    FOREIGN KEY (profile_id) REFERENCES profiles (id) ON DELETE CASCADE
);
