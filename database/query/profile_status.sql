
-- name: UpsertRuleEvaluations :one
INSERT INTO rule_evaluations (
    profile_id, repository_id, artifact_id, pull_request_id, rule_type_id, entity, rule_name
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (profile_id, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID), entity, rule_type_id, lower(rule_name))
  DO UPDATE SET profile_id = $1
RETURNING id;

-- name: UpsertRuleDetailsEval :one
INSERT INTO rule_details_eval (
    rule_eval_id,
    status,
    details,
    last_updated
)
VALUES ($1, $2, $3, NOW())
 ON CONFLICT(rule_eval_id)
    DO UPDATE SET
           status = $2,
           details = $3,
           last_updated = NOW()
    WHERE rule_details_eval.rule_eval_id = $1
RETURNING id;

-- name: UpsertRuleDetailsRemediate :one
INSERT INTO rule_details_remediate (
    rule_eval_id,
    status,
    details,
    metadata,
    last_updated
)
VALUES ($1, $2, $3, sqlc.arg(metadata)::jsonb, NOW())
ON CONFLICT(rule_eval_id)
    DO UPDATE SET
                  status = $2,
                  details = $3,
                  metadata = sqlc.arg(metadata)::jsonb,
                  last_updated = NOW()
    WHERE rule_details_remediate.rule_eval_id = $1
RETURNING id;

-- name: UpsertRuleDetailsAlert :one
INSERT INTO rule_details_alert (
    rule_eval_id,
    status,
    details,
    metadata,
    last_updated
)
VALUES ($1, $2, $3, sqlc.arg(metadata)::jsonb, NOW())
ON CONFLICT(rule_eval_id)
    DO UPDATE SET
                  status = $2,
                  details = $3,
                  metadata = sqlc.arg(metadata)::jsonb,
                  last_updated = NOW()
    WHERE rule_details_alert.rule_eval_id = $1
RETURNING id;

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

-- DeleteRuleStatusesForProfileAndRuleType deletes a rule evaluation
-- but locks the table before doing so.

-- name: DeleteRuleStatusesForProfileAndRuleType :exec
DELETE FROM rule_evaluations
WHERE id IN (
    SELECT id FROM rule_evaluations as re
    WHERE re.profile_id = $1 AND re.rule_type_id = $2 AND re.rule_name = $3 FOR UPDATE);

-- name: ListRuleEvaluationsByProfileId :many
WITH
   eval_details AS (
   SELECT
       rule_eval_id,
       status AS eval_status,
       details AS eval_details,
       last_updated AS eval_last_updated
   FROM rule_details_eval
   ),
   remediation_details AS (
       SELECT
           rule_eval_id,
           status AS rem_status,
           details AS rem_details,
           metadata AS rem_metadata,
           last_updated AS rem_last_updated
       FROM rule_details_remediate
   ),
   alert_details AS (
       SELECT
           rule_eval_id,
           status AS alert_status,
           details AS alert_details,
           metadata AS alert_metadata,
           last_updated AS alert_last_updated
       FROM rule_details_alert
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
    res.id AS rule_evaluation_id,
    res.repository_id,
    res.entity,
    res.rule_name,
    repo.repo_name,
    repo.repo_owner,
    repo.provider,
    rt.name AS rule_type_name,
    rt.severity_value as rule_type_severity_value,
    rt.id AS rule_type_id,
    rt.guidance as rule_type_guidance
FROM rule_evaluations res
         LEFT JOIN eval_details ed ON ed.rule_eval_id = res.id
         LEFT JOIN remediation_details rd ON rd.rule_eval_id = res.id
         LEFT JOIN alert_details ad ON ad.rule_eval_id = res.id
         INNER JOIN repositories repo ON repo.id = res.repository_id
         INNER JOIN rule_type rt ON rt.id = res.rule_type_id
WHERE res.profile_id = $1 AND
    (
        CASE
            WHEN sqlc.narg(entity_type)::entities = 'repository' AND res.repository_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_type)::entities  = 'artifact' AND res.artifact_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_type)::entities  = 'artifact' AND res.artifact_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_type)::entities  = 'pull_request' AND res.pull_request_id = sqlc.narg(entity_id)::UUID THEN true
            WHEN sqlc.narg(entity_id)::UUID IS NULL THEN true
            ELSE false
            END
        ) AND (rt.name = sqlc.narg(rule_type_name) OR sqlc.narg(rule_type_name) IS NULL)
          AND (lower(res.rule_name) = lower(sqlc.narg(rule_name)) OR sqlc.narg(rule_name) IS NULL)
;
