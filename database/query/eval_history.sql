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

-- name: GetLatestEvalStateForRuleEntity :one
SELECT eh.* FROM evaluation_rule_entities AS re
JOIN latest_evaluation_statuses AS les ON les.rule_entity_id = re.id
JOIN evaluation_statuses AS eh ON les.evaluation_history_id = eh.id
WHERE re.rule_id = $1
AND (
    re.repository_id = $2
    OR re.pull_request_id = $3
    OR re.artifact_id = $4
)
FOR UPDATE;

-- name: InsertEvaluationRuleEntity :one
INSERT INTO evaluation_rule_entities(
    rule_id,
    repository_id,
    pull_request_id,
    artifact_id
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING id;

-- name: InsertEvaluationStatus :one
INSERT INTO evaluation_statuses(
    rule_entity_id,
    status,
    details
) VALUES (
    $1,
    $2,
    $3
)
RETURNING id;

-- name: UpdateEvaluationTimes :exec
UPDATE evaluation_statuses
SET
    evaluation_times = $1,
    most_recent_evaluation = NOW()
WHERE id = $2;

-- name: UpsertLatestEvaluationStatus :exec
INSERT INTO latest_evaluation_statuses(
    rule_entity_id,
    evaluation_history_id
) VALUES (
    $1,
    $2
)
ON CONFLICT (rule_entity_id, evaluation_history_id) DO UPDATE
SET evaluation_history_id = $2;