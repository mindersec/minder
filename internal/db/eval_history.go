// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/google/uuid"
)

type ListEvaluationHistoryParams struct {
	Next            *time.Time               `json:"next"`
	Prev            *time.Time               `json:"prev"`
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
	Fromts          *time.Time               `json:"fromts"`
	Tots            *time.Time               `json:"tots"`
	Size            uint                     `json:"size"`
}

type ListEvaluationHistoryRow struct {
	EvaluationID       uuid.UUID                  `json:"evaluation_id"`
	EvaluatedAt        time.Time                  `json:"evaluated_at"`
	EntityType         interface{}                `json:"entity_type"`
	EntityID           interface{}                `json:"entity_id"`
	EntityName         interface{}                `json:"entity_name"`
	RuleType           string                     `json:"rule_type"`
	RuleName           string                     `json:"rule_name"`
	ProfileName        string                     `json:"profile_name"`
	EvaluationStatus   EvalStatusTypes            `json:"evaluation_status"`
	EvaluationDetails  string                     `json:"evaluation_details"`
	RemediationStatus  NullRemediationStatusTypes `json:"remediation_status"`
	RemediationDetails sql.NullString             `json:"remediation_details"`
	AlertStatus        NullAlertStatusTypes       `json:"alert_status"`
	AlertDetails       sql.NullString             `json:"alert_details"`
}

func (q *Queries) ListEvaluationHistory(ctx context.Context, params ListEvaluationHistoryParams) ([]ListEvaluationHistoryRow, error) {
	where := fromParams(&params)
	sql, args, err := goqu.Dialect("postgres").
		Select(
			goqu.I("evaluation_statuses.id").As("evaluation_id"),
			goqu.I("evaluation_statuses.most_recent_evaluation").As("evaluated_at"),
			goqu.Case().
				When(goqu.I("evaluation_rule_entities.repository_id").IsNotNull(), goqu.L("'repository'::entities")).
				When(goqu.I("evaluation_rule_entities.pull_request_id").IsNotNull(), goqu.L("'pull_request'::entities")).
				When(goqu.I("evaluation_rule_entities.artifact_id").IsNotNull(), goqu.L("'artifact'::entities")).
				As("entity_type"),
			goqu.Case().
				When(goqu.I("evaluation_rule_entities.repository_id").IsNotNull(), goqu.I("repositories.id")).
				When(goqu.I("evaluation_rule_entities.pull_request_id").IsNotNull(), goqu.I("pull_requests.id")).
				When(goqu.I("evaluation_rule_entities.artifact_id").IsNotNull(), goqu.I("artifacts.id")).
				As("entity_id"),
			goqu.Case().
				When(goqu.I("evaluation_rule_entities.repository_id").IsNotNull(), goqu.I("repositories.repo_name")).
				When(goqu.I("evaluation_rule_entities.pull_request_id").IsNotNull(), goqu.I("pull_requests.pr_number").Cast("text")).
				When(goqu.I("evaluation_rule_entities.artifact_id").IsNotNull(), goqu.I("artifacts.artifact_name")).
				As("entity_name"),
			goqu.I("rule_type.name").As("rule_type"),
			goqu.I("rule_instances.name").As("rule_name"),
			goqu.I("profiles.name").As("profile_name"),
			goqu.I("evaluation_statuses.status").As("evaluation_status"),
			goqu.I("evaluation_statuses.details").As("evaluation_details"),
			goqu.I("remediation_events.status").As("remediation_status"),
			goqu.I("remediation_events.details").As("remediation_details"),
			goqu.I("alert_events.status").As("alert_status"),
			goqu.I("alert_events.details").As("alert_details"),
		).
		From(goqu.I("evaluation_statuses")).
		Join(goqu.I("evaluation_rule_entities"), goqu.On(goqu.I("evaluation_rule_entities.id").Eq(goqu.I("evaluation_statuses.rule_entity_id")))).
		Join(goqu.I("rule_instances"), goqu.On(goqu.I("evaluation_rule_entities.rule_id").Eq(goqu.I("rule_instances.id")))).
		Join(goqu.I("rule_type"), goqu.On(goqu.I("rule_instances.rule_type_id").Eq(goqu.I("rule_type.id")))).
		Join(goqu.I("profiles"), goqu.On(goqu.I("rule_instances.profile_id").Eq(goqu.I("profiles.id")))).
		LeftOuterJoin(goqu.I("repositories"), goqu.On(goqu.I("evaluation_rule_entities.repository_id").Eq(goqu.I("repositories.id")))).
		LeftOuterJoin(goqu.I("pull_requests"), goqu.On(goqu.I("evaluation_rule_entities.pull_request_id").Eq(goqu.I("pull_requests.id")))).
		LeftOuterJoin(goqu.I("artifacts"), goqu.On(goqu.I("evaluation_rule_entities.artifact_id").Eq(goqu.I("artifacts.id")))).
		LeftOuterJoin(goqu.I("remediation_events"), goqu.On(goqu.I("remediation_events.evaluation_id").Eq(goqu.I("evaluation_statuses.id")))).
		LeftOuterJoin(goqu.I("alert_events"), goqu.On(goqu.I("alert_events.evaluation_id").Eq(goqu.I("evaluation_statuses.id")))).
		Where(where).
		Limit(params.Size).
		Prepared(true).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("failed interpolating queries: %w", err)
	}

	stmt, err := q.db.PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.QueryContext(ctx, args...)
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
			&i.EntityName,
			&i.RuleType,
			&i.RuleName,
			&i.ProfileName,
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

func fromParams(
	params *ListEvaluationHistoryParams,
) exp.ExpressionList {
	expression := goqu.And()

	if params.Next != nil {
		expression = expression.Append(goqu.I("evaluation_statuses.most_recent_evaluation").Lt(*params.Next))
	}
	if params.Prev != nil {
		expression = expression.Append(goqu.I("evaluation_statuses.most_recent_evaluation").Gt(*params.Prev))
	}

	if len(params.Entitytypes) != 0 {
		expression = expression.Append(goqu.C("entity_type").In(params.Entitytypes))
	}
	if len(params.Notentitytypes) != 0 {
		expression = expression.Append(goqu.C("entity_type").NotIn(params.Notentitytypes))
	}

	if len(params.Entitynames) != 0 {
		expression = expression.Append(goqu.C("entity_name").In(params.Entitynames))
	}
	if len(params.Notentitynames) != 0 {
		expression = expression.Append(goqu.C("entity_name").In(params.Notentitynames))
	}

	if len(params.Profilenames) != 0 {
		expression = expression.Append(goqu.I("profiles.name").In(params.Profilenames))
	}
	if len(params.Notprofilenames) != 0 {
		expression = expression.Append(goqu.I("profiles.name").NotIn(params.Notprofilenames))
	}

	if len(params.Remediations) != 0 {
		expression = expression.Append(goqu.I("remediation_events.status").In(params.Remediations))
	}
	if len(params.Notremediations) != 0 {
		expression = expression.Append(goqu.I("remediation_events.status").NotIn(params.Notremediations))
	}

	if len(params.Alerts) != 0 {
		expression = expression.Append(goqu.I("alert_events.status").In(params.Alerts))
	}
	if len(params.Notalerts) != 0 {
		expression = expression.Append(goqu.I("alert_events.status").NotIn(params.Notalerts))
	}

	if len(params.Statuses) != 0 {
		expression = expression.Append(goqu.I("evaluation_statuses.status").In(params.Statuses))
	}
	if len(params.Notstatuses) != 0 {
		expression = expression.Append(goqu.I("evaluation_statuses.status").In(params.Notstatuses))
	}

	if params.Fromts != nil && params.Tots != nil {
		expression = expression.Append(goqu.I("evaluation_statuses.most_recent_evaluation").
			Between(goqu.Range(*params.Fromts, *params.Tots)))
	} else {
		if params.Fromts != nil {
			expression = expression.Append(goqu.I("evaluation_statuses.most_recent_evaluation").Gt(*params.Fromts))
		}
		if params.Tots != nil {
			expression = expression.Append(goqu.I("evaluation_statuses.most_recent_evaluation").Lt(*params.Tots))
		}
	}

	return expression
}
