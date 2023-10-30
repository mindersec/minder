
-- name: UpsertRuleEvaluations :one
INSERT INTO rule_evaluations (
    profile_id, repository_id, artifact_id, pull_request_id, rule_type_id, entity
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (profile_id, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID), entity, rule_type_id)
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
    last_updated
)
VALUES ($1, $2, $3, NOW())
ON CONFLICT(rule_eval_id)
    DO UPDATE SET
                  status = $2,
                  details = $3,
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
                  status = CASE WHEN $2 != 'skipped' THEN $2 ELSE rule_details_alert.status END,
                  details = CASE WHEN $2 != 'skipped' THEN $3 ELSE rule_details_alert.details END,
                  metadata = CASE WHEN $2 != 'skipped' THEN sqlc.arg(metadata)::jsonb ELSE rule_details_alert.metadata END,
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
WHERE p.name = $1 AND p.project_id = $2;

-- name: GetProfileStatusByProject :many
SELECT p.id, p.name, ps.profile_status, ps.last_updated FROM profile_status ps
INNER JOIN profiles p ON p.id = ps.profile_id
WHERE p.project_id = $1;

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
    rd.rem_last_updated,
    ad.alert_status,
    ad.alert_details,
    ad.alert_metadata,
    ad.alert_last_updated,
    res.repository_id,
    res.entity,
    repo.repo_name,
    repo.repo_owner,
    repo.provider,
    rt.name AS rule_type_name,
    rt.id AS rule_type_id
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
        ) AND (rt.name = sqlc.narg(rule_name) OR sqlc.narg(rule_name) IS NULL)
;
