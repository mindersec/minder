-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Add rego_version column to rule_type table to cache the detected Rego
-- language version. All existing rule types are assumed to be V0.
ALTER TABLE rule_type ADD COLUMN rego_version TEXT NOT NULL DEFAULT 'v0';
