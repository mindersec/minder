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

// Package history contains logic for tracking evaluation history
package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	propertiessvc "github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/providers/manager"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// EvaluationHistoryService contains methods to add/query data in the history table.
type EvaluationHistoryService interface {
	// StoreEvaluationStatus stores the result of this evaluation in the history table.
	// Returns the UUID of the evaluation status, and the UUID of the rule-entity
	StoreEvaluationStatus(
		ctx context.Context,
		qtx db.Querier,
		ruleID uuid.UUID,
		profileID uuid.UUID,
		entityType db.Entities,
		entityID uuid.UUID,
		evalError error,
		marshaledCheckpoint []byte,
	) (uuid.UUID, error)
	// ListEvaluationHistory returns a list of evaluations stored
	// in the history table.
	ListEvaluationHistory(
		ctx context.Context,
		qtx db.ExtendQuerier,
		cursor *ListEvaluationCursor,
		size uint32,
		filter ListEvaluationFilter,
	) (*ListEvaluationHistoryResult, error)
}

type options func(*evaluationHistoryService)

func withPropertiesServiceBuilder(builder func(qtx db.ExtendQuerier) propertiessvc.PropertiesService) options {
	return func(ehs *evaluationHistoryService) {
		ehs.propServiceBuilder = builder
	}
}

// NewEvaluationHistoryService creates a new instance of EvaluationHistoryService
func NewEvaluationHistoryService(providerManager manager.ProviderManager, opts ...options) EvaluationHistoryService {
	ehs := &evaluationHistoryService{
		providerManager: providerManager,
		propServiceBuilder: func(qtx db.ExtendQuerier) propertiessvc.PropertiesService {
			return propertiessvc.NewPropertiesService(qtx)
		},
	}
	for _, opt := range opts {
		opt(ehs)
	}

	return ehs
}

type evaluationHistoryService struct {
	providerManager    manager.ProviderManager
	propServiceBuilder func(qtx db.ExtendQuerier) propertiessvc.PropertiesService
}

func (e *evaluationHistoryService) StoreEvaluationStatus(
	ctx context.Context,
	qtx db.Querier,
	ruleID uuid.UUID,
	profileID uuid.UUID,
	entityType db.Entities,
	entityID uuid.UUID,
	evalError error,
	marshaledCheckpoint []byte,
) (uuid.UUID, error) {
	var ruleEntityID uuid.UUID
	status := evalerrors.ErrorAsEvalStatus(evalError)
	details := evalerrors.ErrorAsEvalDetails(evalError)

	params := paramsFromEntity(ruleID, entityID)

	// find the latest record for this rule/entity pair
	latestRecord, err := qtx.GetLatestEvalStateForRuleEntity(ctx,
		db.GetLatestEvalStateForRuleEntityParams{
			RuleID:           params.RuleID,
			EntityInstanceID: params.EntityID,
		},
	)
	if err != nil {
		// if we find nothing, create a new rule/entity record
		if errors.Is(err, sql.ErrNoRows) {
			ruleEntityID, err = qtx.InsertEvaluationRuleEntity(ctx,
				db.InsertEvaluationRuleEntityParams{
					RuleID:           params.RuleID,
					EntityType:       entityType,
					EntityInstanceID: params.EntityID,
				},
			)
			if err != nil {
				return uuid.Nil, fmt.Errorf("error while creating new rule/entity in database: %w", err)
			}
		} else {
			return uuid.Nil, fmt.Errorf("error while querying DB: %w", err)
		}
	} else {
		ruleEntityID = latestRecord.RuleEntityID
	}

	evaluationID, err := e.createNewStatus(ctx, qtx, ruleEntityID, profileID, status, details, marshaledCheckpoint)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error while creating new evaluation status for rule/entity %s: %w", ruleEntityID, err)
	}

	return evaluationID, nil
}

func (_ *evaluationHistoryService) createNewStatus(
	ctx context.Context,
	qtx db.Querier,
	ruleEntityID uuid.UUID,
	profileID uuid.UUID,
	status db.EvalStatusTypes,
	details string,
	marshaledCheckpoint []byte,
) (uuid.UUID, error) {
	newEvaluationID, err := qtx.InsertEvaluationStatus(ctx,
		db.InsertEvaluationStatusParams{
			RuleEntityID: ruleEntityID,
			Status:       status,
			Details:      details,
			Checkpoint:   marshaledCheckpoint,
		},
	)
	if err != nil {
		return uuid.Nil, err
	}

	// mark this as the latest status for this rule/entity
	err = qtx.UpsertLatestEvaluationStatus(ctx,
		db.UpsertLatestEvaluationStatusParams{
			RuleEntityID:        ruleEntityID,
			EvaluationHistoryID: newEvaluationID,
			ProfileID:           profileID,
		},
	)
	if err != nil {
		return uuid.Nil, err
	}

	return newEvaluationID, err
}

func paramsFromEntity(
	ruleID uuid.UUID,
	entityID uuid.UUID,
) *ruleEntityParams {
	params := ruleEntityParams{RuleID: ruleID}

	params.EntityID = entityID
	return &params
}

type ruleEntityParams struct {
	RuleID   uuid.UUID
	EntityID uuid.UUID
}

func (ehs *evaluationHistoryService) ListEvaluationHistory(
	ctx context.Context,
	qtx db.ExtendQuerier,
	cursor *ListEvaluationCursor,
	size uint32,
	filter ListEvaluationFilter,
) (*ListEvaluationHistoryResult, error) {
	params := db.ListEvaluationHistoryParams{
		Size: int64(size),
	}

	if err := toSQLCursor(cursor, &params); err != nil {
		return nil, err
	}
	if err := toSQLFilter(filter, &params); err != nil {
		return nil, err
	}

	rows, err := qtx.ListEvaluationHistory(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("error listing history: %w", err)
	}

	if cursor != nil && cursor.Direction == Prev {
		slices.Reverse(rows)
	}

	propsvc := ehs.propServiceBuilder(qtx)

	data := make([]*OneEvalHistoryAndEntity, 0, len(rows))
	for _, row := range rows {
		efp, err := propsvc.EntityWithProperties(ctx, row.EntityID, qtx)
		if err != nil {
			return nil, fmt.Errorf("error fetching entity for properties: %w", err)
		}

		err = propsvc.RetrieveAllPropertiesForEntity(ctx, efp, ehs.providerManager)
		if err != nil {
			return nil, fmt.Errorf("error fetching properties for entity: %w", err)
		}

		data = append(data, &OneEvalHistoryAndEntity{
			EntityWithProperties: efp,
			EvalHistoryRow:       row,
		})
	}

	result := &ListEvaluationHistoryResult{
		Data: data,
	}
	if len(rows) > 0 {
		newest := rows[0]
		oldest := rows[len(rows)-1]

		result.Next = []byte(fmt.Sprintf("+%d", oldest.EvaluatedAt.UnixMicro()))
		result.Prev = []byte(fmt.Sprintf("-%d", newest.EvaluatedAt.UnixMicro()))
	}

	return result, nil
}

func toSQLCursor(
	cursor *ListEvaluationCursor,
	params *db.ListEvaluationHistoryParams,
) error {
	if cursor == nil {
		return nil
	}

	switch cursor.Direction {
	case Next:
		params.Next = sql.NullTime{
			Time:  cursor.Time,
			Valid: true,
		}
	case Prev:
		params.Prev = sql.NullTime{
			Time:  cursor.Time,
			Valid: true,
		}
	default:
		return fmt.Errorf(
			"invalid cursor direction: %s",
			string(cursor.Direction),
		)
	}

	return nil
}

func toSQLFilter(
	filter ListEvaluationFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	if filter == nil {
		return nil
	}

	if err := paramsFromProjectFilter(filter, params); err != nil {
		return err
	}
	if err := paramsFromEntityTypeFilter(filter, params); err != nil {
		return err
	}
	if err := paramsFromEntityNameFilter(filter, params); err != nil {
		return err
	}
	if err := paramsFromProfileNameFilter(filter, params); err != nil {
		return err
	}
	if err := paramsFromRemediationFilter(filter, params); err != nil {
		return err
	}
	if err := paramsFromAlertFilter(filter, params); err != nil {
		return err
	}
	if err := paramsFromStatusFilter(filter, params); err != nil {
		return err
	}
	return paramsFromTimeRangeFilter(filter, params)
}

func paramsFromProjectFilter(
	filter ProjectFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	params.Projectid = filter.GetProjectID()
	return nil
}

func paramsFromEntityTypeFilter(
	filter EntityTypeFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	if len(filter.IncludedEntityTypes()) != 0 {
		entityTypes, err := convert(
			filter.IncludedEntityTypes(),
			mapEntities,
		)
		if err != nil {
			return err
		}
		params.Entitytypes = entityTypes
	}
	if len(filter.ExcludedEntityTypes()) != 0 {
		entityTypes, err := convert(
			filter.ExcludedEntityTypes(),
			mapEntities,
		)
		if err != nil {
			return fmt.Errorf("error filtering entity types: %w", err)
		}
		params.Notentitytypes = entityTypes
	}
	return nil
}

func paramsFromEntityNameFilter(
	filter EntityNameFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	if len(filter.IncludedEntityNames()) != 0 {
		params.Entitynames = filter.IncludedEntityNames()
	}
	if len(filter.ExcludedEntityNames()) != 0 {
		params.Notentitynames = filter.ExcludedEntityNames()
	}
	return nil
}

func paramsFromProfileNameFilter(
	filter ProfileNameFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	if len(filter.IncludedProfileNames()) != 0 {
		params.Profilenames = filter.IncludedProfileNames()
	}
	if len(filter.ExcludedProfileNames()) != 0 {
		params.Notprofilenames = filter.ExcludedProfileNames()
	}
	return nil
}

func paramsFromRemediationFilter(
	filter RemediationFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	if len(filter.IncludedRemediations()) != 0 {
		remediations, err := convert(
			filter.IncludedRemediations(),
			mapRemediationStatusTypes,
		)
		if err != nil {
			return err
		}
		params.Remediations = remediations
	}
	if len(filter.ExcludedRemediations()) != 0 {
		remediations, err := convert(
			filter.ExcludedRemediations(),
			mapRemediationStatusTypes,
		)
		if err != nil {
			return err
		}
		params.Notremediations = remediations
	}
	return nil
}

func paramsFromAlertFilter(
	filter AlertFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	if len(filter.IncludedAlerts()) != 0 {
		alerts, err := convert(
			filter.IncludedAlerts(),
			mapAlertStatusTypes,
		)
		if err != nil {
			return fmt.Errorf("error filtering alerts: %w", err)
		}
		params.Alerts = alerts
	}
	if len(filter.ExcludedAlerts()) != 0 {
		alerts, err := convert(
			filter.ExcludedAlerts(),
			mapAlertStatusTypes,
		)
		if err != nil {
			return err
		}
		params.Notalerts = alerts
	}
	return nil
}

func paramsFromStatusFilter(
	filter StatusFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	if len(filter.IncludedStatuses()) != 0 {
		statuses, err := convert(
			filter.IncludedStatuses(),
			mapEvalStatusTypes,
		)
		if err != nil {
			return err
		}
		params.Statuses = statuses
	}
	if len(filter.ExcludedStatuses()) != 0 {
		statuses, err := convert(
			filter.ExcludedStatuses(),
			mapEvalStatusTypes,
		)
		if err != nil {
			return err
		}
		params.Notstatuses = statuses
	}
	return nil
}

func paramsFromTimeRangeFilter(
	filter TimeRangeFilter,
	params *db.ListEvaluationHistoryParams,
) error {
	if filter.GetFrom() != nil {
		params.Fromts = sql.NullTime{
			Time:  *filter.GetFrom(),
			Valid: true,
		}
	}
	if filter.GetTo() != nil {
		params.Tots = sql.NullTime{
			Time:  *filter.GetTo(),
			Valid: true,
		}
	}
	return nil
}

func convert[
	T db.Entities |
		db.RemediationStatusTypes |
		db.AlertStatusTypes |
		db.EvalStatusTypes,
](
	values []string,
	mapf func(string) (T, error),
) ([]T, error) {
	converted := make([]T, 0, len(values))
	for _, v := range values {
		dbObj, err := mapf(v)
		if err != nil {
			return nil, err
		}
		converted = append(converted, dbObj)
	}
	return converted, nil
}

func mapEntities(value string) (db.Entities, error) {
	switch value {
	case "repository":
		return db.EntitiesRepository, nil
	case "build_environment":
		return db.EntitiesBuildEnvironment, nil
	case "artifact":
		return db.EntitiesArtifact, nil
	case "pull_request":
		return db.EntitiesPullRequest, nil
	default:
		return db.Entities("invalid"),
			fmt.Errorf("invalid entity: %s", value)
	}
}

//nolint:goconst
func mapRemediationStatusTypes(
	value string,
) (db.RemediationStatusTypes, error) {
	switch value {
	case "success":
		return db.RemediationStatusTypesSuccess, nil
	case "failure":
		return db.RemediationStatusTypesFailure, nil
	case "error":
		return db.RemediationStatusTypesError, nil
	case "skipped":
		return db.RemediationStatusTypesSkipped, nil
	case "not_available":
		return db.RemediationStatusTypesNotAvailable, nil
	case "pending":
		return db.RemediationStatusTypesPending, nil
	default:
		return db.RemediationStatusTypes("invalid"),
			fmt.Errorf("invalid remediation status: %s", value)
	}
}

//nolint:goconst
func mapAlertStatusTypes(
	value string,
) (db.AlertStatusTypes, error) {
	switch value {
	case "on":
		return db.AlertStatusTypesOn, nil
	case "off":
		return db.AlertStatusTypesOff, nil
	case "error":
		return db.AlertStatusTypesError, nil
	case "skipped":
		return db.AlertStatusTypesSkipped, nil
	case "not_available":
		return db.AlertStatusTypesNotAvailable, nil
	default:
		return db.AlertStatusTypes("invalid"),
			fmt.Errorf("invalid alert status: %s", value)
	}
}

//nolint:goconst
func mapEvalStatusTypes(
	value string,
) (db.EvalStatusTypes, error) {
	switch value {
	case "success":
		return db.EvalStatusTypesSuccess, nil
	case "failure":
		return db.EvalStatusTypesFailure, nil
	case "error":
		return db.EvalStatusTypesError, nil
	case "skipped":
		return db.EvalStatusTypesSkipped, nil
	case "pending":
		return db.EvalStatusTypesPending, nil
	default:
		return db.EvalStatusTypes("invalid"),
			fmt.Errorf("invalid evaluation status: %s", value)
	}
}
