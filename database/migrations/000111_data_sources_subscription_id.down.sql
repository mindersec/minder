-- SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE rule_type DROP COLUMN subscription_id;

COMMIT;
