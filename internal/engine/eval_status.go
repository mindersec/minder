// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func (e *Executor) createEvalStatusParams(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	profile *pb.Profile,
	rule *pb.Profile_Rule,
) (*engif.EvalStatusParams, error) {
	// Get Profile UUID
	profileID, err := uuid.Parse(*profile.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing profile ID: %w", err)
	}

	repoID, artID, prID := inf.GetEntityDBIDs()

	params := &engif.EvalStatusParams{
		Rule:          rule,
		Profile:       profile,
		ProfileID:     profileID,
		EntityType:    entities.EntityTypeToDB(inf.Type),
		RepoID:        repoID,
		ArtifactID:    artID,
		PullRequestID: prID,
	}

	// Prepare params for fetching the current rule evaluation from the database
	entityType := db.NullEntities{
		Entities: params.EntityType,
		Valid:    true}
	entityID := uuid.NullUUID{}
	switch params.EntityType {
	case db.EntitiesArtifact:
		entityID = params.ArtifactID
	case db.EntitiesRepository:
		entityID = uuid.NullUUID{
			UUID:  params.RepoID,
			Valid: true,
		}
	case db.EntitiesPullRequest:
		entityID = params.PullRequestID
	case db.EntitiesBuildEnvironment:
		return nil, fmt.Errorf("build environment entity type not supported")
	}

	ruleTypeName := sql.NullString{
		String: rule.Type,
		Valid:  true,
	}

	ruleName := sql.NullString{
		String: rule.Name,
		Valid:  true,
	}

	// Get the current rule evaluation from the database
	evalStatus, err := e.querier.GetRuleEvaluationByProfileIdAndRuleType(ctx,
		params.ProfileID,
		entityType,
		ruleName,
		entityID,
		ruleTypeName,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting rule evaluation status from db: %w", err)
	}

	// Save the current rule evaluation status to the evalParams
	params.EvalStatusFromDb = &evalStatus

	return params, nil
}

func (e *Executor) createOrUpdateEvalStatus(
	ctx context.Context,
	params *engif.EvalStatusParams,
) error {
	logger := params.DecorateLogger(zerolog.Ctx(ctx).With().Logger())
	// Make sure evalParams is not nil
	if params == nil {
		return fmt.Errorf("createEvalStatusParams cannot be nil")
	}

	// Check if we should skip silently
	if errors.Is(params.GetEvalErr(), evalerrors.ErrEvaluationSkipSilently) {
		logger.Info().Msg("rule evaluation skipped silently - skip updating the database")
		return nil
	}

	// Upsert evaluation
	id, err := e.querier.UpsertRuleEvaluations(ctx, db.UpsertRuleEvaluationsParams{
		ProfileID: params.ProfileID,
		RepositoryID: uuid.NullUUID{
			UUID:  params.RepoID,
			Valid: true,
		},
		ArtifactID:    params.ArtifactID,
		Entity:        params.EntityType,
		RuleTypeID:    params.RuleTypeID,
		PullRequestID: params.PullRequestID,
		RuleName:      params.Rule.Name,
	})

	if err != nil {
		logger.Err(err).Msg("error upserting rule evaluation")
		return err
	}

	// Upsert evaluation details
	_, err = e.querier.UpsertRuleDetailsEval(ctx, db.UpsertRuleDetailsEvalParams{
		RuleEvalID: id,
		Status:     evalerrors.ErrorAsEvalStatus(params.GetEvalErr()),
		Details:    evalerrors.ErrorAsEvalDetails(params.GetEvalErr()),
	})

	if err != nil {
		logger.Err(err).Msg("error upserting rule evaluation details")
		return err
	}
	// Upsert remediation details
	_, err = e.querier.UpsertRuleDetailsRemediate(ctx, db.UpsertRuleDetailsRemediateParams{
		RuleEvalID: id,
		Status:     evalerrors.ErrorAsRemediationStatus(params.GetActionsErr().RemediateErr),
		Details:    errorAsActionDetails(params.GetActionsErr().RemediateErr),
		Metadata:   params.GetActionsErr().RemediateMeta,
	})
	if err != nil {
		logger.Err(err).Msg("error upserting rule remediation details")
	}
	// Upsert alert details
	_, err = e.querier.UpsertRuleDetailsAlert(ctx, db.UpsertRuleDetailsAlertParams{
		RuleEvalID: id,
		Status:     evalerrors.ErrorAsAlertStatus(params.GetActionsErr().AlertErr),
		Details:    errorAsActionDetails(params.GetActionsErr().AlertErr),
		Metadata:   params.GetActionsErr().AlertMeta,
	})
	if err != nil {
		logger.Err(err).Msg("error upserting rule alert details")
	}
	return err
}

func errorAsActionDetails(err error) string {
	if evalerrors.IsActionFatalError(err) {
		return err.Error()
	}

	return ""
}
