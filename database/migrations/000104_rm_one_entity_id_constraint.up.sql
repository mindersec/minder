-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE evaluation_rule_entities DROP CONSTRAINT one_entity_id;

COMMIT;
