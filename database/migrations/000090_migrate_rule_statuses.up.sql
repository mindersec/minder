-- Copyright 2024 Stacklok, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

BEGIN;

-- Since we introduced the evaluation history tables, we dual write
-- evaluation statuses to both the evaluation history tables, as well
-- as the old tables which only keep track of the latest state of rule
-- evaluations. In order to fully replace the old tables, we need to migrate
-- any rule evaluations which happened before we started dual writing.
--
-- As of the previous migration, any entry in rule_evaluations which has a
-- corresponding entry in evaluation_rule_entities will have the `migrated`
-- column set to `TRUE`. In order to simplify the migration this transaction
-- will start off by populating a temporary table with the IDs of the
-- rule_evaluation rows which we need to migrate.
--
-- This table also has an evaluation_status_id which will be populated later.
-- This is needed because remediations and alerts in the evaluation history
-- tables are linked by foreign key to the evaluation_statuses table instead
-- of the evaluation_rule_entities table.
CREATE TEMPORARY TABLE temp_migrate_rule_evaluations(
    rule_evaluation_id UUID NOT NULL,
    evaluation_status_id UUID DEFAULT NULL
)
ON COMMIT DROP;

-- Populate the temporary table with the IDs of the rule_evaluation rows we
-- need to migrate.
INSERT INTO temp_migrate_rule_evaluations (rule_evaluation_id)
SELECT id FROM rule_evaluations AS re
WHERE re.migrated = FALSE;

-- Insert the missing evaluations into evaluation_rule_entities.
--
-- Note that we reuse the PK ID from rule_evaluations as the PK for
-- evaluation_rule_entities. The other tables we need to migrate have FK
-- references to rule_evaluations, so by reusing the same PK ID, we can
-- simplify some of the subsequent queries.
--
-- In an ideal world, we could have a single query which just copies the three
-- entity IDs. However, the rule_evaluations table sometimes has a repository_id
-- for a non-repo entity. Since evaluation_rule_entities has the constraint that
-- only one column is set, copy the rows over type-by-type.
INSERT INTO evaluation_rule_entities (id, rule_id, pull_request_id, entity_type)
SELECT id, rule_instance_id AS rule_id, pull_request_id, entity AS entity_type
FROM rule_evaluations AS re
WHERE re.id IN (SELECT rule_evaluation_id FROM temp_migrate_rule_evaluations)
AND re.entity = 'pull_request'::entities;

INSERT INTO evaluation_rule_entities (id, rule_id, artifact_id, entity_type)
SELECT id, rule_instance_id AS rule_id, artifact_id, entity AS entity_type
FROM rule_evaluations AS re
WHERE re.id IN (SELECT rule_evaluation_id FROM temp_migrate_rule_evaluations)
AND re.entity = 'artifact'::entities;

INSERT INTO evaluation_rule_entities (id, rule_id, repository_id, entity_type)
SELECT id, rule_instance_id AS rule_id, repository_id, entity AS entity_type
FROM rule_evaluations AS re
WHERE re.id IN (SELECT rule_evaluation_id FROM temp_migrate_rule_evaluations)
AND re.entity = 'repository'::entities;

-- Migrate the rule details into evaluation_statuses.
INSERT INTO evaluation_statuses (rule_entity_id, status, details, evaluation_time)
SELECT tmp.rule_evaluation_id, rde.status, rde.details, rde.last_updated
FROM temp_migrate_rule_evaluations AS tmp
JOIN rule_details_eval AS rde ON rde.rule_eval_id = tmp.rule_evaluation_id;

-- At this point, we populate the evaluation_status_id column in the temporary
-- table.
UPDATE temp_migrate_rule_evaluations AS tmp
SET evaluation_status_id = es.id
FROM evaluation_statuses AS es
WHERE es.rule_entity_id = tmp.rule_evaluation_id;

-- Populate the latest_evaluation_history tables with the new evaluation_statuses rows.
INSERT INTO latest_evaluation_statuses (rule_entity_id, evaluation_history_id, profile_id)
SELECT tmp.rule_evaluation_id, tmp.evaluation_status_id, ri.profile_id
FROM temp_migrate_rule_evaluations AS tmp
JOIN evaluation_rule_entities AS ere ON tmp.rule_evaluation_id = ere.id
JOIN rule_instances AS ri ON ere.rule_id = ri.id;

-- Migrate the remediations.
INSERT INTO remediation_events (evaluation_id, status, details, metadata, created_at)
SELECT
    es.id AS evaluation_id, rdr.status AS status, rdr.details AS details,
    rdr.metadata AS metadata, es.evaluation_time AS created_at
FROM rule_details_remediate AS rdr
JOIN temp_migrate_rule_evaluations AS tmp ON tmp.rule_evaluation_id = rdr.rule_eval_id
JOIN evaluation_statuses AS es ON es.id = tmp.evaluation_status_id;

-- Finally migrate the alerts.
INSERT INTO alert_events (evaluation_id, status, details, metadata, created_at)
SELECT
    es.id AS evaluation_id, rda.status AS status, rda.details AS details,
    rda.metadata AS metadata, es.evaluation_time AS created_at
FROM rule_details_alert AS rda
JOIN temp_migrate_rule_evaluations AS tmp ON tmp.rule_evaluation_id = rda.rule_eval_id
JOIN evaluation_statuses AS es ON es.id = tmp.evaluation_status_id;

-- Set the migrated column in the rows we migrated from
-- rule_evaluations.
UPDATE rule_evaluations
SET migrated = TRUE
WHERE id IN (SELECT rule_evaluation_id FROM temp_migrate_rule_evaluations);

-- this is not strictly necessary, but if we don't do it - sqlc will generate
-- a model for the temporary table :(
DROP TABLE IF EXISTS temp_migrate_rule_evaluations;

COMMIT;