-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Add personal user details to users table
ALTER TABLE users
    ADD COLUMN email TEXT,
    ADD COLUMN first_name TEXT,
    ADD COLUMN last_name TEXT;
