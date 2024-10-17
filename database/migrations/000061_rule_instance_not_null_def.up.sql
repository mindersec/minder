-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE rule_instances ALTER COLUMN def SET NOT NULL;
ALTER TABLE rule_instances ALTER COLUMN params SET NOT NULL;

COMMIT;
