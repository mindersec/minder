-- name: GetProfileStatusByIdAndProject :one
SELECT p.id, p.name, ps.profile_status, ps.last_updated FROM profile_status ps
INNER JOIN profiles p ON p.id = ps.profile_id
WHERE p.id = $1 AND p.project_id = $2;

-- name: GetProfileStatusByNameAndProject :one
SELECT p.id, p.name, ps.profile_status, ps.last_updated FROM profile_status ps
INNER JOIN profiles p ON p.id = ps.profile_id
WHERE lower(p.name) = lower(sqlc.arg(name)) AND p.project_id = $1;

-- name: GetProfileStatusByProject :many
SELECT p.id, p.name, ps.profile_status, ps.last_updated FROM profile_status ps
INNER JOIN profiles p ON p.id = ps.profile_id
WHERE p.project_id = $1;

-- ListOldestRuleEvaluationsByRepositoryId has casts in select statement as sqlc generates incorrect types.
-- cast after MIN is required due to a known bug in sqlc: https://github.com/sqlc-dev/sqlc/issues/1965

-- name: ListOldestRuleEvaluationsByRepositoryId :many
SELECT ere.repository_id::uuid AS repository_id, MIN(es.evaluation_time)::timestamp AS oldest_last_updated
FROM evaluation_rule_entities AS ere
    INNER JOIN latest_evaluation_statuses AS les ON ere.id = les.rule_entity_id
    INNER JOIN evaluation_statuses AS es ON les.evaluation_history_id = es.id
WHERE ere.repository_id = ANY (sqlc.arg('repository_ids')::uuid[])
GROUP BY ere.repository_id;

-- name: ListRuleEvaluationsByProfileId :many
WITH
   eval_details AS (
   SELECT
       id,
       status AS eval_status,
       details AS eval_details,
       evaluation_time AS eval_last_updated
   FROM evaluation_statuses
   ),
   remediation_details AS (
       SELECT
           evaluation_id,
           status AS rem_status,
           details AS rem_details,
           metadata AS rem_metadata,
           created_at AS rem_last_updated
       FROM remediation_events
   ),
   alert_details AS (
       SELECT
           evaluation_id,
           status AS alert_status,
           details AS alert_details,
           metadata AS alert_metadata,
           created_at AS alert_last_updated
       FROM alert_events
   )

SELECT
    ed.eval_status,
    ed.eval_last_updated,
    ed.eval_details,
    rd.rem_status,
    rd.rem_details,
    rd.rem_metadata,
    rd.rem_last_updated,
    ad.alert_status,
    ad.alert_details,
    ad.alert_metadata,
    ad.alert_last_updated,
    ed.id AS rule_evaluation_id,
    ere.repository_id,
    ere.entity_type,
    ri.name AS rule_name,
    repo.repo_name,
    repo.repo_owner,
    repo.provider,
    rt.name AS rule_type_name,
    rt.severity_value as rule_type_severity_value,
    rt.id AS rule_type_id,
    rt.guidance as rule_type_guidance,
    rt.display_name as rule_type_display_name,
    -- TODO: store entity ID directly in evaluation_rule_entities
    CASE
        WHEN ere.entity_type = 'artifact'::entities THEN ere.artifact_id
        WHEN ere.entity_type = 'repository'::entities THEN ere.repository_id
        WHEN ere.entity_type = 'pull_request'::entities THEN ere.pull_request_id
    END::uuid as entity_id,
    rt.release_phase as rule_type_release_phase
FROM latest_evaluation_statuses les
         INNER JOIN evaluation_rule_entities ere ON ere.id = les.rule_entity_id
         INNER JOIN eval_details ed ON ed.id = les.evaluation_history_id
         INNER JOIN remediation_details rd ON rd.evaluation_id = les.evaluation_history_id
         INNER JOIN alert_details ad ON ad.evaluation_id = les.evaluation_history_id
         INNER JOIN rule_instances AS ri ON ri.id = ere.rule_id
         INNER JOIN rule_type rt ON rt.id = ri.rule_type_id
         LEFT JOIN repositories repo ON repo.id = ere.repository_id
WHERE les.profile_id = $1 AND
    (
        CASE
            WHEN sqlc.narg(entity_type)::entities = 'repository' AND ere.repository_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_type)::entities  = 'artifact' AND ere.artifact_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_type)::entities  = 'pull_request' AND ere.pull_request_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_id)::UUID IS NULL THEN true
            ELSE false
            END
        ) AND (rt.name = sqlc.narg(rule_type_name) OR sqlc.narg(rule_type_name) IS NULL)
          AND (lower(ri.name) = lower(sqlc.narg(rule_name)) OR sqlc.narg(rule_name) IS NULL)
;
