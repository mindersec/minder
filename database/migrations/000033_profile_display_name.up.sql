-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Adds a `display_name` column to the `profiles` table. The display name defaults
-- to the profile's `name` column, but can be overridden by the user.
ALTER TABLE profiles ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
