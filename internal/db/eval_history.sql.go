// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: eval_history.sql

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const deleteEvaluationHistoryByIDs = `-- name: DeleteEvaluationHistoryByIDs :execrows
DELETE FROM evaluation_statuses s
 WHERE s.id = ANY($1::uuid[])
`

func (q *Queries) DeleteEvaluationHistoryByIDs(ctx context.Context, evaluationids []uuid.UUID) (int64, error) {
	result, err := q.db.ExecContext(ctx, deleteEvaluationHistoryByIDs, pq.Array(evaluationids))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

const getEvaluationHistory = `-- name: GetEvaluationHistory :one
SELECT s.id::uuid AS evaluation_id,
    s.evaluation_time as evaluated_at,
    ere.entity_type,
    -- entity id
    ere.entity_instance_id as entity_id,
    -- entity name
    ei.name as entity_name,
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
    JOIN entity_instances ei ON ere.entity_instance_id = ei.id
    JOIN projects j ON ei.project_id = j.id
    LEFT JOIN remediation_events re ON re.evaluation_id = s.id
    LEFT JOIN alert_events ae ON ae.evaluation_id = s.id
WHERE s.id = $1 AND j.id = $2
`

type GetEvaluationHistoryParams struct {
	EvaluationID uuid.UUID `json:"evaluation_id"`
	ProjectID    uuid.UUID `json:"project_id"`
}

type GetEvaluationHistoryRow struct {
	EvaluationID       uuid.UUID                  `json:"evaluation_id"`
	EvaluatedAt        time.Time                  `json:"evaluated_at"`
	EntityType         Entities                   `json:"entity_type"`
	EntityID           uuid.UUID                  `json:"entity_id"`
	EntityName         string                     `json:"entity_name"`
	ProjectID          uuid.UUID                  `json:"project_id"`
	RuleType           string                     `json:"rule_type"`
	RuleName           string                     `json:"rule_name"`
	RuleSeverity       Severity                   `json:"rule_severity"`
	ProfileName        string                     `json:"profile_name"`
	EvaluationStatus   EvalStatusTypes            `json:"evaluation_status"`
	EvaluationDetails  string                     `json:"evaluation_details"`
	RemediationStatus  NullRemediationStatusTypes `json:"remediation_status"`
	RemediationDetails sql.NullString             `json:"remediation_details"`
	AlertStatus        NullAlertStatusTypes       `json:"alert_status"`
	AlertDetails       sql.NullString             `json:"alert_details"`
}

func (q *Queries) GetEvaluationHistory(ctx context.Context, arg GetEvaluationHistoryParams) (GetEvaluationHistoryRow, error) {
	row := q.db.QueryRowContext(ctx, getEvaluationHistory, arg.EvaluationID, arg.ProjectID)
	var i GetEvaluationHistoryRow
	err := row.Scan(
		&i.EvaluationID,
		&i.EvaluatedAt,
		&i.EntityType,
		&i.EntityID,
		&i.EntityName,
		&i.ProjectID,
		&i.RuleType,
		&i.RuleName,
		&i.RuleSeverity,
		&i.ProfileName,
		&i.EvaluationStatus,
		&i.EvaluationDetails,
		&i.RemediationStatus,
		&i.RemediationDetails,
		&i.AlertStatus,
		&i.AlertDetails,
	)
	return i, err
}

const getLatestEvalStateForRuleEntity = `-- name: GetLatestEvalStateForRuleEntity :one

SELECT eh.id, eh.rule_entity_id, eh.status, eh.details, eh.evaluation_time, eh.checkpoint FROM evaluation_rule_entities AS re
JOIN latest_evaluation_statuses AS les ON les.rule_entity_id = re.id
JOIN evaluation_statuses AS eh ON les.evaluation_history_id = eh.id
WHERE re.rule_id = $1 AND re.entity_instance_id = $2
FOR UPDATE
`

type GetLatestEvalStateForRuleEntityParams struct {
	RuleID           uuid.UUID `json:"rule_id"`
	EntityInstanceID uuid.UUID `json:"entity_instance_id"`
}

// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0
func (q *Queries) GetLatestEvalStateForRuleEntity(ctx context.Context, arg GetLatestEvalStateForRuleEntityParams) (EvaluationStatus, error) {
	row := q.db.QueryRowContext(ctx, getLatestEvalStateForRuleEntity, arg.RuleID, arg.EntityInstanceID)
	var i EvaluationStatus
	err := row.Scan(
		&i.ID,
		&i.RuleEntityID,
		&i.Status,
		&i.Details,
		&i.EvaluationTime,
		&i.Checkpoint,
	)
	return i, err
}

const insertAlertEvent = `-- name: InsertAlertEvent :exec
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
)
`

type InsertAlertEventParams struct {
	EvaluationID uuid.UUID        `json:"evaluation_id"`
	Status       AlertStatusTypes `json:"status"`
	Details      string           `json:"details"`
	Metadata     json.RawMessage  `json:"metadata"`
}

func (q *Queries) InsertAlertEvent(ctx context.Context, arg InsertAlertEventParams) error {
	_, err := q.db.ExecContext(ctx, insertAlertEvent,
		arg.EvaluationID,
		arg.Status,
		arg.Details,
		arg.Metadata,
	)
	return err
}

const insertEvaluationRuleEntity = `-- name: InsertEvaluationRuleEntity :one
INSERT INTO evaluation_rule_entities(
    rule_id,
    entity_type,
    entity_instance_id
) VALUES (
    $1,
    $2,
    $3
)
RETURNING id
`

type InsertEvaluationRuleEntityParams struct {
	RuleID           uuid.UUID `json:"rule_id"`
	EntityType       Entities  `json:"entity_type"`
	EntityInstanceID uuid.UUID `json:"entity_instance_id"`
}

func (q *Queries) InsertEvaluationRuleEntity(ctx context.Context, arg InsertEvaluationRuleEntityParams) (uuid.UUID, error) {
	row := q.db.QueryRowContext(ctx, insertEvaluationRuleEntity, arg.RuleID, arg.EntityType, arg.EntityInstanceID)
	var id uuid.UUID
	err := row.Scan(&id)
	return id, err
}

const insertEvaluationStatus = `-- name: InsertEvaluationStatus :one
INSERT INTO evaluation_statuses(
    rule_entity_id,
    status,
    details,
    checkpoint
) VALUES (
    $1,
    $2,
    $3,
    $4::jsonb
)
RETURNING id
`

type InsertEvaluationStatusParams struct {
	RuleEntityID uuid.UUID       `json:"rule_entity_id"`
	Status       EvalStatusTypes `json:"status"`
	Details      string          `json:"details"`
	Checkpoint   json.RawMessage `json:"checkpoint"`
}

func (q *Queries) InsertEvaluationStatus(ctx context.Context, arg InsertEvaluationStatusParams) (uuid.UUID, error) {
	row := q.db.QueryRowContext(ctx, insertEvaluationStatus,
		arg.RuleEntityID,
		arg.Status,
		arg.Details,
		arg.Checkpoint,
	)
	var id uuid.UUID
	err := row.Scan(&id)
	return id, err
}

const insertRemediationEvent = `-- name: InsertRemediationEvent :exec
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
)
`

type InsertRemediationEventParams struct {
	EvaluationID uuid.UUID              `json:"evaluation_id"`
	Status       RemediationStatusTypes `json:"status"`
	Details      string                 `json:"details"`
	Metadata     json.RawMessage        `json:"metadata"`
}

func (q *Queries) InsertRemediationEvent(ctx context.Context, arg InsertRemediationEventParams) error {
	_, err := q.db.ExecContext(ctx, insertRemediationEvent,
		arg.EvaluationID,
		arg.Status,
		arg.Details,
		arg.Metadata,
	)
	return err
}

const listEvaluationHistory = `-- name: ListEvaluationHistory :many
SELECT s.id::uuid AS evaluation_id,
       s.evaluation_time as evaluated_at,
       ere.entity_type,
       -- entity id
        ere.entity_instance_id as entity_id,
       j.id as project_id,
       -- rule type, name, and profile
       rt.name AS rule_type,
       ri.name AS rule_name,
       rt.severity_value as rule_severity,
       p.name AS profile_name,
       p.labels as profile_labels,
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
  JOIN entity_instances ei ON ere.entity_instance_id = ei.id
  JOIN projects j ON ei.project_id = j.id
  LEFT JOIN remediation_events re ON re.evaluation_id = s.id
  LEFT JOIN alert_events ae ON ae.evaluation_id = s.id
 WHERE ($1::timestamp without time zone IS NULL OR $1 > s.evaluation_time)
   AND ($2::timestamp without time zone IS NULL OR $2 < s.evaluation_time)
   -- inclusion filters
   AND ($3::entities[] IS NULL OR ere.entity_type = ANY($3::entities[]))
   AND ($4::text[] IS NULL OR ei.name = ANY($4::text[]))
   AND ($5::text[] IS NULL OR p.name = ANY($5::text[]))
   AND ($6::remediation_status_types[] IS NULL OR re.status = ANY($6::remediation_status_types[]))
   AND ($7::alert_status_types[] IS NULL OR ae.status = ANY($7::alert_status_types[]))
   AND ($8::eval_status_types[] IS NULL OR s.status = ANY($8::eval_status_types[]))
   -- exclusion filters
   AND ($9::entities[] IS NULL OR ere.entity_type != ALL($9::entities[]))
   AND ($10::text[] IS NULL OR ei.name != ALL($10::text[]))
   AND ($11::text[] IS NULL OR p.name != ALL($11::text[]))
   AND ($12::remediation_status_types[] IS NULL OR re.status != ALL($12::remediation_status_types[]))
   AND ($13::alert_status_types[] IS NULL OR ae.status != ALL($13::alert_status_types[]))
   AND ($14::eval_status_types[] IS NULL OR s.status != ALL($14::eval_status_types[]))
   -- time range filter
   AND ($15::timestamp without time zone IS NULL OR s.evaluation_time >= $15)
   AND ($16::timestamp without time zone IS NULL OR  s.evaluation_time < $16)
   -- implicit filter by project id
   AND j.id = $17
   -- implicit filter by profile labels
   AND (($18::text[] IS NULL AND p.labels = array[]::text[]) -- include only unlabelled records
	OR (($18::text[] IS NOT NULL AND $18::text[] = array['*']::text[]) -- include all labels
	    OR ($18::text[] IS NOT NULL AND p.labels && $18::text[]) -- include only specified labels
	)
   )
   AND ($19::text[] IS NULL OR NOT p.labels && $19::text[]) -- exclude only specified labels
 ORDER BY
 CASE WHEN $1::timestamp without time zone IS NULL THEN s.evaluation_time END ASC,
 CASE WHEN $2::timestamp without time zone IS NULL THEN s.evaluation_time END DESC
 LIMIT $20::bigint
`

type ListEvaluationHistoryParams struct {
	Next            sql.NullTime             `json:"next"`
	Prev            sql.NullTime             `json:"prev"`
	Entitytypes     []Entities               `json:"entitytypes"`
	Entitynames     []string                 `json:"entitynames"`
	Profilenames    []string                 `json:"profilenames"`
	Remediations    []RemediationStatusTypes `json:"remediations"`
	Alerts          []AlertStatusTypes       `json:"alerts"`
	Statuses        []EvalStatusTypes        `json:"statuses"`
	Notentitytypes  []Entities               `json:"notentitytypes"`
	Notentitynames  []string                 `json:"notentitynames"`
	Notprofilenames []string                 `json:"notprofilenames"`
	Notremediations []RemediationStatusTypes `json:"notremediations"`
	Notalerts       []AlertStatusTypes       `json:"notalerts"`
	Notstatuses     []EvalStatusTypes        `json:"notstatuses"`
	Fromts          sql.NullTime             `json:"fromts"`
	Tots            sql.NullTime             `json:"tots"`
	Projectid       uuid.UUID                `json:"projectid"`
	Labels          []string                 `json:"labels"`
	Notlabels       []string                 `json:"notlabels"`
	Size            int64                    `json:"size"`
}

type ListEvaluationHistoryRow struct {
	EvaluationID       uuid.UUID                  `json:"evaluation_id"`
	EvaluatedAt        time.Time                  `json:"evaluated_at"`
	EntityType         Entities                   `json:"entity_type"`
	EntityID           uuid.UUID                  `json:"entity_id"`
	ProjectID          uuid.UUID                  `json:"project_id"`
	RuleType           string                     `json:"rule_type"`
	RuleName           string                     `json:"rule_name"`
	RuleSeverity       Severity                   `json:"rule_severity"`
	ProfileName        string                     `json:"profile_name"`
	ProfileLabels      []string                   `json:"profile_labels"`
	EvaluationStatus   EvalStatusTypes            `json:"evaluation_status"`
	EvaluationDetails  string                     `json:"evaluation_details"`
	RemediationStatus  NullRemediationStatusTypes `json:"remediation_status"`
	RemediationDetails sql.NullString             `json:"remediation_details"`
	AlertStatus        NullAlertStatusTypes       `json:"alert_status"`
	AlertDetails       sql.NullString             `json:"alert_details"`
}

func (q *Queries) ListEvaluationHistory(ctx context.Context, arg ListEvaluationHistoryParams) ([]ListEvaluationHistoryRow, error) {
	rows, err := q.db.QueryContext(ctx, listEvaluationHistory,
		arg.Next,
		arg.Prev,
		pq.Array(arg.Entitytypes),
		pq.Array(arg.Entitynames),
		pq.Array(arg.Profilenames),
		pq.Array(arg.Remediations),
		pq.Array(arg.Alerts),
		pq.Array(arg.Statuses),
		pq.Array(arg.Notentitytypes),
		pq.Array(arg.Notentitynames),
		pq.Array(arg.Notprofilenames),
		pq.Array(arg.Notremediations),
		pq.Array(arg.Notalerts),
		pq.Array(arg.Notstatuses),
		arg.Fromts,
		arg.Tots,
		arg.Projectid,
		pq.Array(arg.Labels),
		pq.Array(arg.Notlabels),
		arg.Size,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListEvaluationHistoryRow{}
	for rows.Next() {
		var i ListEvaluationHistoryRow
		if err := rows.Scan(
			&i.EvaluationID,
			&i.EvaluatedAt,
			&i.EntityType,
			&i.EntityID,
			&i.ProjectID,
			&i.RuleType,
			&i.RuleName,
			&i.RuleSeverity,
			&i.ProfileName,
			pq.Array(&i.ProfileLabels),
			&i.EvaluationStatus,
			&i.EvaluationDetails,
			&i.RemediationStatus,
			&i.RemediationDetails,
			&i.AlertStatus,
			&i.AlertDetails,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listEvaluationHistoryStaleRecords = `-- name: ListEvaluationHistoryStaleRecords :many
SELECT s.evaluation_time,
       s.id,
       ere.rule_id,
       -- entity type
       ere.entity_type,
       -- entity id
       ere.entity_instance_id as entity_id
  FROM evaluation_statuses s
       JOIN evaluation_rule_entities ere ON s.rule_entity_id = ere.id
       LEFT JOIN latest_evaluation_statuses l
	   ON l.rule_entity_id = s.rule_entity_id
	   AND l.evaluation_history_id = s.id
 WHERE s.evaluation_time < $1
  -- the following predicate ensures we get only "stale" records
   AND l.evaluation_history_id IS NULL
 -- listing from oldest to newest
 ORDER BY s.evaluation_time ASC, rule_id ASC, entity_id ASC
 LIMIT $2::integer
`

type ListEvaluationHistoryStaleRecordsParams struct {
	Threshold time.Time `json:"threshold"`
	Size      int32     `json:"size"`
}

type ListEvaluationHistoryStaleRecordsRow struct {
	EvaluationTime time.Time `json:"evaluation_time"`
	ID             uuid.UUID `json:"id"`
	RuleID         uuid.UUID `json:"rule_id"`
	EntityType     Entities  `json:"entity_type"`
	EntityID       uuid.UUID `json:"entity_id"`
}

func (q *Queries) ListEvaluationHistoryStaleRecords(ctx context.Context, arg ListEvaluationHistoryStaleRecordsParams) ([]ListEvaluationHistoryStaleRecordsRow, error) {
	rows, err := q.db.QueryContext(ctx, listEvaluationHistoryStaleRecords, arg.Threshold, arg.Size)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListEvaluationHistoryStaleRecordsRow{}
	for rows.Next() {
		var i ListEvaluationHistoryStaleRecordsRow
		if err := rows.Scan(
			&i.EvaluationTime,
			&i.ID,
			&i.RuleID,
			&i.EntityType,
			&i.EntityID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const upsertLatestEvaluationStatus = `-- name: UpsertLatestEvaluationStatus :exec
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
SET evaluation_history_id = $2
`

type UpsertLatestEvaluationStatusParams struct {
	RuleEntityID        uuid.UUID `json:"rule_entity_id"`
	EvaluationHistoryID uuid.UUID `json:"evaluation_history_id"`
	ProfileID           uuid.UUID `json:"profile_id"`
}

func (q *Queries) UpsertLatestEvaluationStatus(ctx context.Context, arg UpsertLatestEvaluationStatusParams) error {
	_, err := q.db.ExecContext(ctx, upsertLatestEvaluationStatus, arg.RuleEntityID, arg.EvaluationHistoryID, arg.ProfileID)
	return err
}
