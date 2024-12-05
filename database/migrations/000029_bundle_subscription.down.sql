-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE projects DROP COLUMN subscription_id;
ALTER TABLE rule_type DROP COLUMN subscription_id;

DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS bundles;
