-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE evaluation_rule_entities ALTER COLUMN entity_type DROP NOT NULL;
