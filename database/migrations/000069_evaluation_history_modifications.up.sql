-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- these changes reflect feedback during design review plus some changes I've
-- wanted to make after writing queries around these tables.

-- give some of the tables better names
-- these tables are not yet used in the codebase, so renaming them should be low risk
ALTER TABLE rule_entity_evaluations RENAME TO evaluation_rule_entities;
ALTER TABLE evaluation_history      RENAME TO evaluation_statuses;
ALTER TABLE latest_evaluation_state RENAME to latest_evaluation_statuses;

-- ensure FK cascades are set for entities, and for the alerts/remediations
ALTER TABLE evaluation_rule_entities DROP CONSTRAINT rule_entity_evaluations_artifact_id_fkey;
ALTER TABLE evaluation_rule_entities DROP CONSTRAINT rule_entity_evaluations_repository_id_fkey;
ALTER TABLE evaluation_rule_entities DROP CONSTRAINT rule_entity_evaluations_pull_request_id_fkey;
ALTER TABLE remediation_events DROP CONSTRAINT remediation_events_evaluation_id_fkey;
ALTER TABLE alert_events DROP CONSTRAINT alert_events_evaluation_id_fkey;

ALTER TABLE evaluation_rule_entities ADD FOREIGN KEY (artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE;
ALTER TABLE evaluation_rule_entities ADD FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE;
ALTER TABLE evaluation_rule_entities ADD FOREIGN KEY (pull_request_id) REFERENCES pull_requests(id) ON DELETE CASCADE;
ALTER TABLE remediation_events ADD FOREIGN KEY (evaluation_id) REFERENCES evaluation_statuses(id) ON DELETE CASCADE;
ALTER TABLE alert_events ADD FOREIGN KEY (evaluation_id) REFERENCES evaluation_statuses(id) ON DELETE CASCADE;

-- use an array of timestamps instead of a separate table when tracking evaluation instances
DROP TABLE IF EXISTS evaluation_instance;
ALTER TABLE evaluation_statuses ADD COLUMN evaluation_times TIMESTAMP[] NOT NULL DEFAULT ARRAY[NOW()]::TIMESTAMP[];

COMMIT;
