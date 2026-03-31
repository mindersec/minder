// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package commit_status provides necessary interfaces and implementations for
// processing pull request commit status check alerts.
package commit_status

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/interfaces"
	pbinternal "github.com/mindersec/minder/internal/proto"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// AlertType is the type of the commit status alert engine
	AlertType = "commit_status"
)

// Alert is the structure backing the commit status alert
type Alert struct {
	actionType interfaces.ActionType
	gh         provifv1.GitHub
	alertCfg   *pb.RuleType_Definition_Alert_AlertTypeCommitStatus
	setting    models.ActionOpt
}

type paramsPR struct {
	Owner      string
	Repo       string
	CommitSha  string
	Number     int
	Metadata   *alertMetadata
	prevStatus *db.ListRuleEvaluationsByProfileIdRow
}

type alertMetadata struct {
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
}

// NewCommitStatusAlert creates a new commit status alert action
func NewCommitStatusAlert(
	actionType interfaces.ActionType,
	alertCfg *pb.RuleType_Definition_Alert_AlertTypeCommitStatus,
	gh provifv1.GitHub,
	setting models.ActionOpt,
) (*Alert, error) {
	if actionType == "" {
		return nil, fmt.Errorf("action type cannot be empty")
	}

	return &Alert{
		actionType: actionType,
		gh:         gh,
		alertCfg:   alertCfg,
		setting:    setting,
	}, nil
}

// Class returns the action type of the commit status alert engine
func (alert *Alert) Class() interfaces.ActionType {
	return alert.actionType
}

// Type returns the action subtype of the commit status alert engine
func (*Alert) Type() string {
	return AlertType
}

// GetOnOffState returns the alert action state read from the profile
func (alert *Alert) GetOnOffState() models.ActionOpt {
	return models.ActionOptOrDefault(alert.setting, models.ActionOptOff)
}

// Do sets the commit status on a pull request
func (alert *Alert) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	entity protoreflect.ProtoMessage,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (json.RawMessage, error) {
	pr, ok := entity.(*pbinternal.PullRequest)
	if !ok {
		return nil, fmt.Errorf("expected pull request, got %T", entity)
	}

	commitStatusParams, err := alert.getParamsForCommitStatus(ctx, pr, params, metadata)
	if err != nil {
		return nil, fmt.Errorf("error extracting parameters for commit status: %w", err)
	}

	// Process the command based on the action setting
	switch alert.setting {
	case models.ActionOptOn:
		return alert.run(ctx, commitStatusParams, cmd, params)
	case models.ActionOptDryRun:
		return alert.runDry(ctx, commitStatusParams, cmd, params)
	case models.ActionOptOff, models.ActionOptUnknown:
		return nil, fmt.Errorf("unexpected action setting: %w", enginerr.ErrActionFailed)
	}
	return nil, enginerr.ErrActionSkipped
}

func (alert *Alert) run(
	ctx context.Context,
	params *paramsPR,
	cmd interfaces.ActionCmd,
	actionParams interfaces.ActionsParams,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx)

	// Context string for the commit status check (e.g. "minder/rule_name")
	// If rule_name isn't available, we fallback to "minder"
	contextStr := "minder"
	if rule := actionParams.GetRule(); rule != nil {
		contextStr = fmt.Sprintf("minder/%s", rule.Name)
	}

	switch cmd {
	// ActionCmdOn: The evaluation failed. Set status to failure.
	case interfaces.ActionCmdOn:
		commitStatus := &github.RepoStatus{
			Context:     github.String(contextStr),
			State:       github.String("failure"),
			Description: github.String("Minder evaluation failed"),
		}

		_, err := alert.gh.SetCommitStatus(
			ctx,
			params.Owner,
			params.Repo,
			params.CommitSha,
			commitStatus,
		)
		if err != nil {
			logger.Error().Err(err).Msg("error setting commit status")
			return nil, enginerr.ErrActionFailed
		}

		now := time.Now()
		newMeta, err := json.Marshal(alertMetadata{
			SubmittedAt: &now,
		})
		if err != nil {
			return nil, fmt.Errorf("error marshalling alert metadata json: %w", err)
		}

		logger.Info().Str("commit_sha", params.CommitSha).Msg("PR commit status updated to failure")
		return newMeta, nil

	// ActionCmdOff: The evaluation succeeded (or alert turned off). Set status to success.
	case interfaces.ActionCmdOff:
		commitStatus := &github.RepoStatus{
			Context:     github.String(contextStr),
			State:       github.String("success"),
			Description: github.String("Minder evaluation succeeded"),
		}

		_, err := alert.gh.SetCommitStatus(
			ctx,
			params.Owner,
			params.Repo,
			params.CommitSha,
			commitStatus,
		)
		if err != nil {
			logger.Error().Err(err).Msg("error setting commit status")
		}

		logger.Info().Str("commit_sha", params.CommitSha).Msg("PR commit status updated to success")
		// Return ErrActionTurnedOff to indicate the action resolved appropriately
		return nil, fmt.Errorf("%s: %w", alert.Class(), enginerr.ErrActionTurnedOff)

	case interfaces.ActionCmdDoNothing:
		// Return the previous alert status.
		return alert.runDoNothing(ctx, params)
	}
	return nil, enginerr.ErrActionSkipped
}

// runDry runs the commit status action in dry run mode, logging what it would do
func (alert *Alert) runDry(
	ctx context.Context,
	params *paramsPR,
	cmd interfaces.ActionCmd,
	actionParams interfaces.ActionsParams,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx)

	contextStr := "minder"
	if rule := actionParams.GetRule(); rule != nil {
		contextStr = fmt.Sprintf("minder/%s", rule.Name)
	}

	switch cmd {
	case interfaces.ActionCmdOn:
		logger.Info().Msgf("dry run: set commit status to failure for context %s on PR %d in repo %s/%s",
			contextStr, params.Number, params.Owner, params.Repo)
		return nil, nil
	case interfaces.ActionCmdOff:
		logger.Info().Msgf("dry run: set commit status to success for context %s on PR %d in repo %s/%s",
			contextStr, params.Number, params.Owner, params.Repo)
		return nil, nil
	case interfaces.ActionCmdDoNothing:
		return alert.runDoNothing(ctx, params)
	}
	return nil, enginerr.ErrActionSkipped
}

// runDoNothing returns the previous alert status
func (*Alert) runDoNothing(ctx context.Context, params *paramsPR) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().Str("repo", params.Repo).Logger()
	logger.Debug().Msg("Running do nothing")

	err := enginerr.AlertStatusAsError(params.prevStatus)
	if params.prevStatus != nil {
		return params.prevStatus.AlertMetadata, err
	}
	return nil, err
}

// getParamsForCommitStatus extracts the details from the entity
func (alert *Alert) getParamsForCommitStatus(
	ctx context.Context,
	pr *pbinternal.PullRequest,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (*paramsPR, error) {
	logger := zerolog.Ctx(ctx)
	result := &paramsPR{
		prevStatus: params.GetEvalStatusFromDb(),
		Owner:      pr.GetRepoOwner(),
		Repo:       pr.GetRepoName(),
		CommitSha:  pr.GetCommitSha(),
	}

	if pr.Number > int64(math.MaxInt32) {
		return nil, fmt.Errorf("pr number is too large")
	}
	result.Number = int(pr.Number)

	if metadata != nil {
		meta := &alertMetadata{}
		err := json.Unmarshal(*metadata, meta)
		if err != nil {
			logger.Debug().Msgf("error unmarshalling alert metadata: %v", err)
		} else {
			result.Metadata = meta
		}
	}

	return result, nil
}
