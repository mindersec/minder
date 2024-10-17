-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- severity is an enum that represents the severity of a rule
CREATE TYPE severity AS ENUM ('unknown', 'info', 'low', 'medium', 'high', 'critical');

ALTER TABLE rule_type ADD COLUMN severity_value severity NOT NULL DEFAULT 'unknown';
