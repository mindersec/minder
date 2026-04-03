// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package commit_status provides necessary interfaces and implementations for
// processing pull request commit status check alerts.
package commit_status

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog"
	uritemplate "github.com/std-uritemplate/std-uritemplate/go/v2"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/interfaces"
	pbinternal "github.com/mindersec/minder/internal/proto"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// AlertType is the type of the commit status alert engine
	AlertType = "commit_status"
	// DescriptionMaxLength is the maximum length of the commit status description
	DescriptionMaxLength = 1024
)

// Alert is the structure backing the commit status alert
type Alert struct {
	actionType interfaces.ActionType
	gh         provifv1.CommitStatusPublisher
	alertCfg   *pb.RuleType_Definition_Alert_AlertTypeCommitStatus
	setting    models.ActionOpt
}

// CommitStatusTemplateParams is the parameters for the commit status templates
type CommitStatusTemplateParams struct {
	// EvalErrorDetails is the details of the error that occurred during evaluation, which may be empty
	EvalErrorDetails string

	// EvalResult is the output of the evaluation, which may be empty
	EvalResultOutput any

	// RuleName is the name of the rule
	RuleName string
}

type paramsPR struct {
	Owner       string
	Repo        string
	CommitSha   string
	Number      int
	Context     string
	Description string
	TargetURL   string
	Metadata    *alertMetadata
	prevStatus  *db.ListRuleEvaluationsByProfileIdRow
}

type alertMetadata struct {
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
}

// NewCommitStatusAlert creates a new commit status alert action
func NewCommitStatusAlert(
	actionType interfaces.ActionType,
	alertCfg *pb.RuleType_Definition_Alert_AlertTypeCommitStatus,
	gh provifv1.CommitStatusPublisher,
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
	ctx context.Context, params *paramsPR,
	cmd interfaces.ActionCmd, _ interfaces.ActionsParams,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx)

	switch cmd {
	// ActionCmdOn: The evaluation failed. Set status to failure.
	case interfaces.ActionCmdOn:
		desc := params.Description
		if desc == "" {
			desc = "Minder evaluation failed"
		}
		commitStatus := &provifv1.CommitStatus{
			Context:     params.Context,
			State:       provifv1.CommitStatusFailure,
			Description: desc,
			TargetURL:   params.TargetURL,
		}
		if err := alert.gh.PublishCommitStatus(ctx, params.Owner, params.Repo, params.CommitSha, commitStatus); err != nil {
			return nil, fmt.Errorf("error setting commit status: %w, %w", err, enginerr.ErrActionFailed)
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
		desc := params.Description
		if desc == "" {
			desc = "Minder evaluation succeeded"
		}
		commitStatus := &provifv1.CommitStatus{
			Context:     params.Context,
			State:       provifv1.CommitStatusSuccess,
			Description: desc,
			TargetURL:   params.TargetURL,
		}
		if err := alert.gh.PublishCommitStatus(ctx, params.Owner, params.Repo, params.CommitSha, commitStatus); err != nil {
			return nil, fmt.Errorf("error dismissing commit status: %w, %w", err, enginerr.ErrActionFailed)
		}

		logger.Info().Str("commit_sha", params.CommitSha).Msg("PR commit status updated to success")
		// Return ErrActionTurnedOff to indicate the action resolved appropriately
		return nil, fmt.Errorf("%s : %w", alert.Class(), enginerr.ErrActionTurnedOff)

	case interfaces.ActionCmdDoNothing:
		// Return the previous alert status.
		return alert.runDoNothing(ctx, params)
	}
	return nil, enginerr.ErrActionSkipped
}

// runDry runs the commit status action in dry run mode, logging what it would do
func (alert *Alert) runDry(
	ctx context.Context, params *paramsPR,
	cmd interfaces.ActionCmd, _ interfaces.ActionsParams,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx)

	switch cmd {
	case interfaces.ActionCmdOn:
		logger.Info().Msgf("dry run: set commit status to failure for context %s on PR %d in repo %s/%s",
			params.Context, params.Number, params.Owner, params.Repo)
		return nil, nil
	case interfaces.ActionCmdOff:
		logger.Info().Msgf("dry run: set commit status to success for context %s on PR %d in repo %s/%s",
			params.Context, params.Number, params.Owner, params.Repo)
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

	if pr.Number > math.MaxInt {
		return nil, fmt.Errorf("pr number is too large")
	}
	result.Number = int(pr.Number)

	// Create template params
	tmplParams := &CommitStatusTemplateParams{
		EvalErrorDetails: enginerr.ErrorAsEvalDetails(params.GetEvalErr()),
	}
	if params.GetEvalResult() != nil {
		tmplParams.EvalResultOutput = params.GetEvalResult().Output
	}
	if rule := params.GetRule(); rule != nil {
		tmplParams.RuleName = rule.Name
	}

	// 1. Evaluate context
	contextStr := "minder"
	if tmplParams.RuleName != "" {
		contextStr = fmt.Sprintf("%s/%s", contextStr, tmplParams.RuleName)
	}
	result.Context = cmp.Or(alert.alertCfg.GetStatusName(), contextStr)

	// 2. Evaluate description, using text/template
	descTmplStr := alert.alertCfg.GetDescription()
	if descTmplStr != "" {
		descTmpl, err := util.NewSafeTextTemplate(&descTmplStr, "description")
		if err != nil {
			return nil, fmt.Errorf("cannot parse description template: %w", err)
		}
		descStr, err := descTmpl.Render(ctx, tmplParams, DescriptionMaxLength)
		if err != nil {
			return nil, fmt.Errorf("cannot execute description template: %w", err)
		}
		result.Description = descStr
	}

	// 3. Evaluate target_url, using RFC6570
	targetUrlTmplStr := alert.alertCfg.GetTargetUrl()
	if targetUrlTmplStr != "" {
		argsMap := map[string]any{
			"owner":      result.Owner,
			"repo":       result.Repo,
			"commit_sha": result.CommitSha,
			"rule_name":  tmplParams.RuleName,
			"output":     tmplParams.EvalResultOutput,
		}
		expandedUrl, err := uritemplate.Expand(targetUrlTmplStr, argsMap)
		if err != nil {
			return nil, fmt.Errorf("cannot expand target_url template: %w", err)
		}
		result.TargetURL = expandedUrl
	}

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
