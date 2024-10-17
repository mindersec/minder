-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

DROP TYPE severity;

ALTER TABLE rule_type DROP COLUMN severity_value;
