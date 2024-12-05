-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- This adds a `default_branch` column to the repositories
-- table.

ALTER TABLE repositories ADD COLUMN default_branch text;
