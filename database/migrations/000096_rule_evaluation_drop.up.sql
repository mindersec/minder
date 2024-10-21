-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Drop these tables, they have been replaced by evaluation_rule_entities and the related tables.
DROP TABLE rule_details_eval;
DROP TABLE rule_details_alert;
DROP TABLE rule_details_remediate;
DROP TABLE rule_evaluations;

COMMIT;
