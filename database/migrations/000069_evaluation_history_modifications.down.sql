-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- change tables back to old names
ALTER TABLE evaluation_rule_entities RENAME TO rule_entity_evaluations;
ALTER TABLE evaluation_statuses RENAME TO evaluation_history;
ALTER TABLE latest_evaluation_statuses RENAME TO latest_evaluation_state;

-- recreate FKs without ON DELETE CASCADE
ALTER TABLE rule_entity_evaluations DROP CONSTRAINT evaluation_rule_entities_artifact_id_fkey;
ALTER TABLE rule_entity_evaluations DROP CONSTRAINT evaluation_rule_entities_repository_id_fkey;
ALTER TABLE rule_entity_evaluations DROP CONSTRAINT evaluation_rule_entities_pull_request_id_fkey;
ALTER TABLE remediation_events DROP CONSTRAINT remediation_events_evaluation_id_fkey;
ALTER TABLE alert_events DROP CONSTRAINT alert_events_evaluation_id_fkey;

ALTER TABLE rule_entity_evaluations ADD FOREIGN KEY (artifact_id) REFERENCES artifacts(id);
ALTER TABLE rule_entity_evaluations ADD FOREIGN KEY (repository_id) REFERENCES repositories(id);
ALTER TABLE rule_entity_evaluations ADD FOREIGN KEY (pull_request_id) REFERENCES pull_requests(id);
ALTER TABLE remediation_events ADD FOREIGN KEY (evaluation_id) REFERENCES evaluation_statuses(id);
ALTER TABLE alert_events ADD FOREIGN KEY (evaluation_id) REFERENCES evaluation_statuses(id);

-- recreate evaluation_instance table
ALTER TABLE evaluation_history DROP COLUMN evaluation_times;
CREATE TABLE evaluation_instance(
    evaluation_id UUID NOT NULL REFERENCES evaluation_history(id) ON DELETE CASCADE,
    evaluation_time TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (evaluation_id, evaluation_time)
);

COMMIT;
