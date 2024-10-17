-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- This removes the `default_branch` column from the repositories
-- table.

ALTER TABLE repositories DROP COLUMN default_branch;
