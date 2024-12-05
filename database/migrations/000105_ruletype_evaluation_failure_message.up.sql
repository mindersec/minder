-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Adds a `short_failure_message` column to the `rule_type` table. The failure message
-- is displayed to the user when a rule evaluation fails.
ALTER TABLE rule_type ADD COLUMN short_failure_message TEXT NOT NULL DEFAULT '';
