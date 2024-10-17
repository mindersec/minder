-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- this will be made non-nullable in a future PR
ALTER TABLE evaluation_rule_entities ADD COLUMN entity_type entities;
