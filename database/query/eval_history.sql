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
    artifact_id,
    entity_type
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
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

-- name: UpsertLatestEvaluationStatus :exec
INSERT INTO latest_evaluation_statuses(
    rule_entity_id,
    evaluation_history_id,
    profile_id
) VALUES (
    $1,
    $2,
    $3
)
ON CONFLICT (rule_entity_id) DO UPDATE
SET evaluation_history_id = $2;

-- name: InsertRemediationEvent :exec
INSERT INTO remediation_events(
    evaluation_id,
    status,
    details,
    metadata
) VALUES (
    $1,
    $2,
    $3,
    $4
);

-- name: InsertAlertEvent :exec
INSERT INTO alert_events(
    evaluation_id,
    status,
    details,
    metadata
) VALUES (
    $1,
    $2,
    $3,
    $4
);

-- name: GetEvaluationHistory :one
SELECT s.id::uuid AS evaluation_id,
    s.evaluation_time as evaluated_at,
    ere.entity_type,
    -- entity id
       CAST(
           CASE
               WHEN ere.repository_id IS NOT NULL THEN r.id
               WHEN ere.pull_request_id IS NOT NULL THEN pr.id
               WHEN ere.artifact_id IS NOT NULL THEN a.id
           END AS uuid
       ) AS entity_id,
    -- raw fields for entity names
    r.repo_owner,
    r.repo_name,
    pr.pr_number,
    a.artifact_name,
    j.id as project_id,
    -- rule type, name, and profile
    rt.name AS rule_type,
    ri.name AS rule_name,
    rt.severity_value as rule_severity,
    p.name AS profile_name,
    -- evaluation status and details
    s.status AS evaluation_status,
    s.details AS evaluation_details,
    -- remediation status and details
    re.status AS remediation_status,
    re.details AS remediation_details,
    -- alert status and details
    ae.status AS alert_status,
    ae.details AS alert_details
FROM evaluation_statuses s
    JOIN evaluation_rule_entities ere ON ere.id = s.rule_entity_id
    JOIN rule_instances ri ON ere.rule_id = ri.id
    JOIN rule_type rt ON ri.rule_type_id = rt.id
    JOIN profiles p ON ri.profile_id = p.id
    LEFT JOIN repositories r ON ere.repository_id = r.id
    LEFT JOIN pull_requests pr ON ere.pull_request_id = pr.id
    LEFT JOIN artifacts a ON ere.artifact_id = a.id
    LEFT JOIN remediation_events re ON re.evaluation_id = s.id
    LEFT JOIN alert_events ae ON ae.evaluation_id = s.id
    LEFT JOIN projects j ON r.project_id = j.id
WHERE s.id = sqlc.arg(evaluation_id) AND j.id = sqlc.arg(project_id);

-- name: ListEvaluationHistory :many
SELECT s.id::uuid AS evaluation_id,
       s.evaluation_time as evaluated_at,
       ere.entity_type,
       -- entity id
       CAST(
         CASE
         WHEN ere.repository_id IS NOT NULL THEN r.id
         WHEN ere.pull_request_id IS NOT NULL THEN pr.id
         WHEN ere.artifact_id IS NOT NULL THEN a.id
         END AS uuid
       ) AS entity_id,
       -- raw fields for entity names
       r.repo_owner,
       r.repo_name,
       pr.pr_number,
       a.artifact_name,
       j.id as project_id,
       -- rule type, name, and profile
       rt.name AS rule_type,
       ri.name AS rule_name,
       rt.severity_value as rule_severity,
       p.name AS profile_name,
       -- evaluation status and details
       s.status AS evaluation_status,
       s.details AS evaluation_details,
       -- remediation status and details
       re.status AS remediation_status,
       re.details AS remediation_details,
       -- alert status and details
       ae.status AS alert_status,
       ae.details AS alert_details
  FROM evaluation_statuses s
  JOIN evaluation_rule_entities ere ON ere.id = s.rule_entity_id
  JOIN rule_instances ri ON ere.rule_id = ri.id
  JOIN rule_type rt ON ri.rule_type_id = rt.id
  JOIN profiles p ON ri.profile_id = p.id
  LEFT JOIN repositories r ON ere.repository_id = r.id
  LEFT JOIN pull_requests pr ON ere.pull_request_id = pr.id
  LEFT JOIN artifacts a ON ere.artifact_id = a.id
  LEFT JOIN remediation_events re ON re.evaluation_id = s.id
  LEFT JOIN alert_events ae ON ae.evaluation_id = s.id
  LEFT JOIN projects j ON r.project_id = j.id
 WHERE (sqlc.narg(next)::timestamp without time zone IS NULL OR sqlc.narg(next) > s.evaluation_time)
   AND (sqlc.narg(prev)::timestamp without time zone IS NULL OR sqlc.narg(prev) < s.evaluation_time)
   -- inclusion filters
   AND (sqlc.slice(entityTypes)::entities[] IS NULL OR ere.entity_type = ANY(sqlc.slice(entityTypes)::entities[]))
   AND (sqlc.slice(entityNames)::text[] IS NULL OR ere.repository_id IS NULL OR CONCAT(r.repo_owner, '/', r.repo_name) = ANY(sqlc.slice(entityNames)::text[]))
   AND (sqlc.slice(entityNames)::text[] IS NULL OR ere.pull_request_id IS NULL OR pr.pr_number::text = ANY(sqlc.slice(entityNames)::text[]))
   AND (sqlc.slice(entityNames)::text[] IS NULL OR ere.artifact_id IS NULL OR a.artifact_name = ANY(sqlc.slice(entityNames)::text[]))
   AND (sqlc.slice(profileNames)::text[] IS NULL OR p.name = ANY(sqlc.slice(profileNames)::text[]))
   AND (sqlc.slice(remediations)::remediation_status_types[] IS NULL OR re.status = ANY(sqlc.slice(remediations)::remediation_status_types[]))
   AND (sqlc.slice(alerts)::alert_status_types[] IS NULL OR ae.status = ANY(sqlc.slice(alerts)::alert_status_types[]))
   AND (sqlc.slice(statuses)::eval_status_types[] IS NULL OR s.status = ANY(sqlc.slice(statuses)::eval_status_types[]))
   -- exclusion filters
   AND (sqlc.slice(notEntityTypes)::entities[] IS NULL OR ere.entity_type != ANY(sqlc.slice(notEntityTypes)::entities[]))
   AND (sqlc.slice(notEntityNames)::text[] IS NULL OR ere.repository_id IS NULL OR CONCAT(r.repo_owner, '/', r.repo_name) != ANY(sqlc.slice(notEntityNames)::text[]))
   AND (sqlc.slice(notEntityNames)::text[] IS NULL OR ere.pull_request_id IS NULL OR pr.pr_number::text != ANY(sqlc.slice(notEntityNames)::text[]))
   AND (sqlc.slice(notEntityNames)::text[] IS NULL OR ere.artifact_id IS NULL OR a.artifact_name != ANY(sqlc.slice(notEntityNames)::text[]))
   AND (sqlc.slice(notProfileNames)::text[] IS NULL OR p.name != ANY(sqlc.slice(notProfileNames)::text[]))
   AND (sqlc.slice(notRemediations)::remediation_status_types[] IS NULL OR re.status != ANY(sqlc.slice(notRemediations)::remediation_status_types[]))
   AND (sqlc.slice(notAlerts)::alert_status_types[] IS NULL OR ae.status != ANY(sqlc.slice(notAlerts)::alert_status_types[]))
   AND (sqlc.slice(notStatuses)::eval_status_types[] IS NULL OR s.status != ANY(sqlc.slice(notStatuses)::eval_status_types[]))
   -- time range filter
   AND (sqlc.narg(fromts)::timestamp without time zone IS NULL OR s.evaluation_time >= sqlc.narg(fromts))
   AND (sqlc.narg(tots)::timestamp without time zone IS NULL OR  s.evaluation_time < sqlc.narg(tots))
   -- implicit filter by project id
   AND j.id = sqlc.arg(projectId)
 ORDER BY
 CASE WHEN sqlc.narg(next)::timestamp without time zone IS NULL THEN s.evaluation_time END ASC,
 CASE WHEN sqlc.narg(prev)::timestamp without time zone IS NULL THEN s.evaluation_time END DESC
 LIMIT sqlc.arg(size)::integer;

-- name: ListEvaluationHistoryStaleRecords :many
SELECT s.evaluation_time,
       s.id,
       ere.rule_id,
       -- entity type
       CAST(
	 CASE
	 WHEN ere.repository_id IS NOT NULL THEN 1
	 WHEN ere.pull_request_id IS NOT NULL THEN 2
	 WHEN ere.artifact_id IS NOT NULL THEN 3
	 END AS integer
       ) AS entity_type,
       -- entity id
       CAST(
         CASE
         WHEN ere.repository_id IS NOT NULL THEN ere.repository_id
         WHEN ere.pull_request_id IS NOT NULL THEN ere.pull_request_id
         WHEN ere.artifact_id IS NOT NULL THEN ere.artifact_id
         END AS uuid
       ) AS entity_id
  FROM evaluation_statuses s
       JOIN evaluation_rule_entities ere ON s.rule_entity_id = ere.id
       LEFT JOIN latest_evaluation_statuses l
	   ON l.rule_entity_id = s.rule_entity_id
	   AND l.evaluation_history_id = s.id
 WHERE s.evaluation_time < sqlc.arg(threshold)
  -- the following predicate ensures we get only "stale" records
   AND l.evaluation_history_id IS NULL
 -- listing from oldest to newest
 ORDER BY s.evaluation_time ASC, rule_id ASC, entity_id ASC
 LIMIT sqlc.arg(size)::integer;

-- name: DeleteEvaluationHistoryByIDs :execrows
DELETE FROM evaluation_statuses s
 WHERE s.id = ANY(sqlc.slice(evaluationIds));
