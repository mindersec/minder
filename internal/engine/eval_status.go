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

	"github.com/stacklok/mediator/internal/db"
	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/entities"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func (e *Executor) createEvalStatusParams(
	ctx context.Context,
	inf *EntityInfoWrapper,
	profile *pb.Profile,
	rule *pb.Profile_Rule,
) (*engif.EvalStatusParams, error) {
	// Get Profile UUID
	profileID, err := uuid.Parse(*profile.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing profile ID: %w", err)
	}

	params := &engif.EvalStatusParams{
		Rule:       rule,
		Profile:    profile,
		ProfileID:  profileID,
		RepoID:     uuid.MustParse(inf.OwnershipData[RepositoryIDEventKey]),
		EntityType: entities.EntityTypeToDB(inf.Type),
		ActionsErr: evalerrors.ActionsError{
			RemediateErr: evalerrors.ErrActionSkipped,
			AlertErr:     evalerrors.ErrActionSkipped,
		},
	}

	artifactID, ok := inf.OwnershipData[ArtifactIDEventKey]
	if ok {
		params.ArtifactID = uuid.NullUUID{
			UUID:  uuid.MustParse(artifactID),
			Valid: true,
		}
	}

	pullRequestID, ok := inf.OwnershipData[PullRequestIDEventKey]
	if ok {
		params.PullRequestID = uuid.NullUUID{
			UUID:  uuid.MustParse(pullRequestID),
			Valid: true,
		}
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

	ruleName := sql.NullString{
		String: rule.Type,
		Valid:  true,
	}

	// Get the current rule evaluation from the database
	evalStatus, err := e.querier.GetRuleEvaluationByProfileIdAndRuleType(ctx,
		params.ProfileID,
		entityType,
		entityID,
		ruleName,
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
	evalParams *engif.EvalStatusParams,
) error {
	logger := zerolog.Ctx(ctx)
	// Make sure evalParams is not nil
	if evalParams == nil {
		return fmt.Errorf("createEvalStatusParams cannot be nil")
	}

	// Check if we should skip silently
	if errors.Is(evalParams.EvalErr, evalerrors.ErrEvaluationSkipSilently) {
		logger.Debug().
			Str("repo_id", evalParams.RepoID.String()).
			Str("entity_type", string(evalParams.EntityType)).
			Str("rule_type_id", evalParams.RuleTypeID.String()).
			Str("profile_id", evalParams.ProfileID.String()).
			Msg("rule evaluation skipped silently")
		return nil
	}

	// Upsert evaluation
	id, err := e.querier.UpsertRuleEvaluations(ctx, db.UpsertRuleEvaluationsParams{
		ProfileID: evalParams.ProfileID,
		RepositoryID: uuid.NullUUID{
			UUID:  evalParams.RepoID,
			Valid: true,
		},
		ArtifactID: evalParams.ArtifactID,
		Entity:     evalParams.EntityType,
		RuleTypeID: evalParams.RuleTypeID,
	})

	if err != nil {
		logger.Err(err).
			Str("repo_id", evalParams.RepoID.String()).
			Str("entity_type", string(evalParams.EntityType)).
			Str("profile_id", evalParams.ProfileID.String()).
			Msg("error upserting rule evaluation")
		return err
	}
	// Upsert evaluation details
	_, err = e.querier.UpsertRuleDetailsEval(ctx, db.UpsertRuleDetailsEvalParams{
		RuleEvalID: id,
		Status:     evalerrors.ErrorAsEvalStatus(evalParams.EvalErr),
		Details:    evalerrors.ErrorAsEvalDetails(evalParams.EvalErr),
	})

	if err != nil {
		logger.Err(err).
			Str("repo_id", evalParams.RepoID.String()).
			Str("entity_type", string(evalParams.EntityType)).
			Str("profile_id", evalParams.ProfileID.String()).
			Msg("error upserting rule evaluation details")
		return err
	}
	// Upsert remediation details
	_, err = e.querier.UpsertRuleDetailsRemediate(ctx, db.UpsertRuleDetailsRemediateParams{
		RuleEvalID: id,
		Status:     evalerrors.ErrorAsRemediationStatus(evalParams.ActionsErr.RemediateErr),
		Details:    errorAsActionDetails(evalParams.ActionsErr.RemediateErr),
	})
	if err != nil {
		logger.Err(err).
			Str("repo_id", evalParams.RepoID.String()).
			Str("entity_type", string(evalParams.EntityType)).
			Str("profile_id", evalParams.ProfileID.String()).
			Msg("error upserting rule remediation details")
	}
	// Upsert alert details
	_, err = e.querier.UpsertRuleDetailsAlert(ctx, db.UpsertRuleDetailsAlertParams{
		RuleEvalID: id,
		Status:     evalerrors.ErrorAsAlertStatus(evalParams.ActionsErr.AlertErr),
		Details:    errorAsActionDetails(evalParams.ActionsErr.AlertErr),
		Metadata:   evalParams.ActionsErr.AlertMeta,
	})
	if err != nil {
		logger.Err(err).
			Str("repo_id", evalParams.RepoID.String()).
			Str("entity_type", string(evalParams.EntityType)).
			Str("profile_id", evalParams.ProfileID.String()).
			Msg("error upserting rule alert details")
	}
	return err
}

func errorAsActionDetails(err error) string {
	if evalerrors.IsActionFatalError(err) {
		return err.Error()
	}

	return ""
}
