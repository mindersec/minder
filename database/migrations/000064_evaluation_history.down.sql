-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

DROP TABLE IF EXISTS rule_entity_evaluations;
DROP TABLE IF EXISTS evaluation_history;
DROP TABLE IF EXISTS evaluation_instance;
DROP TABLE IF EXISTS latest_evaluation_state;
DROP TABLE IF EXISTS remediation_events;
DROP TABLE IF EXISTS alert_events;

COMMIT;
