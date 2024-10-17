-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

DROP TABLE rule_instances;
ALTER TABLE entity_profiles DROP COLUMN migrated;

COMMIT;
