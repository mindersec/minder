// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package pull_request_comment provides necessary interfaces and implementations for
// processing pull request comment alerts.
package pull_request_comment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	actionContext "github.com/mindersec/minder/internal/engine/actions/context"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/internal/entities/properties"
	pbinternal "github.com/mindersec/minder/internal/proto"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// AlertType is the type of the pull request comment alert engine
	AlertType = "pull_request_comment"
	// PrCommentMaxLength is the maximum length of the pull request comment
	// (this was derived from the limit of the GitHub API)
	PrCommentMaxLength = 65536
)

// Alert is the structure backing the noop alert
type Alert struct {
	actionType interfaces.ActionType
	commenter  provifv1.PullRequestCommenter
	reviewCfg  *pb.RuleType_Definition_Alert_AlertTypePRComment
	setting    models.ActionOpt
}

// PrCommentTemplateParams is the parameters for the PR comment templates
type PrCommentTemplateParams struct {
	// EvalErrorDetails is the details of the error that occurred during evaluation, which may be empty
	EvalErrorDetails string

	// EvalResult is the output of the evaluation, which may be empty
	EvalResultOutput any
}

type paramsPR struct {
	Comment    string
	props      *properties.Properties
	Metadata   *provifv1.CommentResultMeta
	prevStatus *db.ListRuleEvaluationsByProfileIdRow
}

// NewPullRequestCommentAlert creates a new pull request comment alert action
func NewPullRequestCommentAlert(
	actionType interfaces.ActionType,
	reviewCfg *pb.RuleType_Definition_Alert_AlertTypePRComment,
	gh provifv1.PullRequestCommenter,
	setting models.ActionOpt,
) (*Alert, error) {
	if actionType == "" {
		return nil, fmt.Errorf("action type cannot be empty")
	}

	return &Alert{
		actionType: actionType,
		commenter:  gh,
		reviewCfg:  reviewCfg,
		setting:    setting,
	}, nil
}

// Class returns the action type of the PR comment alert engine
func (alert *Alert) Class() interfaces.ActionType {
	return alert.actionType
}

// Type returns the action subtype of the PR comment alert engine
func (_ *Alert) Type() string {
	return AlertType
}

// GetOnOffState returns the alert action state read from the profile
func (alert *Alert) GetOnOffState() models.ActionOpt {
	return models.ActionOptOrDefault(alert.setting, models.ActionOptOff)
}

// Do comments on a pull request
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

	commentParams, err := alert.getParamsForPRComment(ctx, pr, params, metadata)
	if err != nil {
		return nil, fmt.Errorf("error extracting parameters for PR comment: %w", err)
	}

	// Process the command based on the action setting
	switch alert.setting {
	case models.ActionOptOn:
		return alert.run(ctx, commentParams, cmd)
	case models.ActionOptDryRun:
		return alert.runDry(ctx, commentParams, cmd)
	case models.ActionOptOff, models.ActionOptUnknown:
		return nil, fmt.Errorf("unexpected action setting: %w", enginerr.ErrActionFailed)
	}
	return nil, enginerr.ErrActionSkipped
}

func (alert *Alert) run(ctx context.Context, params *paramsPR, cmd interfaces.ActionCmd) (json.RawMessage, error) {
	// Process the command
	switch cmd {
	case interfaces.ActionCmdOn:
		// Create a review
		return alert.runDoReview(ctx, params)
	case interfaces.ActionCmdOff:
		return json.RawMessage(`{}`), nil
	case interfaces.ActionCmdDoNothing:
		// If the previous status didn't change (still a failure, for instance) we
		// want to refresh the alert.
		if alert.setting == models.ActionOptOn {
			return alert.runDoReview(ctx, params)
		}
		// Else, we just do nothing.
		return alert.runDoNothing(ctx, params)
	}
	return nil, enginerr.ErrActionSkipped
}

// runDry runs the pull request comment action in dry run mode, which logs the comment that would be made
func (alert *Alert) runDry(ctx context.Context, params *paramsPR, cmd interfaces.ActionCmd) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx)

	// Process the command
	switch cmd {
	case interfaces.ActionCmdOn:
		body := github.String(params.Comment)
		logger.Info().Dict("properties", params.props.ToLogDict()).
			Msgf("dry run: create a PR comment on PR with body: %s", *body)
		return nil, nil
	case interfaces.ActionCmdOff:
		if params.Metadata == nil {
			// We cannot do anything without the PR review ID, so we assume that turning the alert off is a success
			return nil, fmt.Errorf("no PR comment ID provided: %w", enginerr.ErrActionTurnedOff)
		}
		logger.Info().Dict("properties", params.props.ToLogDict()).
			Msgf("dry run: dismiss PR comment on PR")
	case interfaces.ActionCmdDoNothing:
		// Return the previous alert status.
		return alert.runDoNothing(ctx, params)

	}
	return nil, enginerr.ErrActionSkipped
}

// runDoNothing returns the previous alert status
func (_ *Alert) runDoNothing(ctx context.Context, params *paramsPR) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().Dict("properties", params.props.ToLogDict()).Logger()

	logger.Debug().Msg("Running do nothing")

	// Return the previous alert status.
	err := enginerr.AlertStatusAsError(params.prevStatus)
	// If there is a valid alert metadata, return it too
	if params.prevStatus != nil {
		return params.prevStatus.AlertMetadata, err
	}
	// If there is no alert metadata, return nil as the metadata and the error
	return nil, err
}

func (alert *Alert) runDoReview(ctx context.Context, params *paramsPR) (json.RawMessage, error) {
	sac := actionContext.GetSharedActionsContext(ctx)
	if sac == nil {
		return nil, fmt.Errorf("shared actions context not found")
	}

	sac.ShareAndRegister("pull_request_comment",
		newAlertFlusher(params.props, params.props.GetProperty(properties.PullRequestCommitSHA).GetString(), alert.commenter),
		&provifv1.PullRequestCommentInfo{
			// TODO: We should add the header to identify the alert. We could use the
			// rule type name.
			Commit: params.props.GetProperty(properties.PullRequestCommitSHA).GetString(),
			Body:   params.Comment,
			// TODO: Determine the priority from the rule type severity
		})
	return json.RawMessage("{}"), nil
}

// getParamsForSecurityAdvisory extracts the details from the entity
func (alert *Alert) getParamsForPRComment(
	ctx context.Context,
	pr *pbinternal.PullRequest,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (*paramsPR, error) {
	logger := zerolog.Ctx(ctx)
	props, err := properties.NewProperties(pr.GetProperties().AsMap())
	if err != nil {
		return nil, fmt.Errorf("error creating properties: %w", err)
	}

	result := &paramsPR{
		prevStatus: params.GetEvalStatusFromDb(),
		props:      props,
	}

	commentTmpl, err := util.NewSafeHTMLTemplate(&alert.reviewCfg.ReviewMessage, "message")
	if err != nil {
		return nil, fmt.Errorf("cannot parse review message template: %w", err)
	}

	tmplParams := &PrCommentTemplateParams{
		EvalErrorDetails: enginerr.ErrorAsEvalDetails(params.GetEvalErr()),
	}

	if params.GetEvalResult() != nil {
		tmplParams.EvalResultOutput = params.GetEvalResult().Output
	}

	comment, err := commentTmpl.Render(ctx, tmplParams, PrCommentMaxLength)
	if err != nil {
		return nil, fmt.Errorf("cannot execute title template: %w", err)
	}

	result.Comment = comment

	// Unmarshal the existing alert metadata, if any
	if metadata != nil {
		meta := &provifv1.CommentResultMeta{}
		err := json.Unmarshal(*metadata, meta)
		if err != nil {
			// There's nothing saved apparently, so no need to fail here, but do log the error
			logger.Debug().Msgf("error unmarshalling alert metadata: %v", err)
		} else {
			result.Metadata = meta
		}
	}

	return result, nil
}

type alertFlusher struct {
	props     *properties.Properties
	commitSha string
	commenter provifv1.PullRequestCommenter
}

func newAlertFlusher(props *properties.Properties, commitSha string, commenter provifv1.PullRequestCommenter) *alertFlusher {
	return &alertFlusher{
		props:     props,
		commitSha: commitSha,
		commenter: commenter,
	}
}

func (a *alertFlusher) Flush(ctx context.Context, items ...any) error {
	logger := zerolog.Ctx(ctx)

	var aggregatedCommentBody string

	// iterate and aggregate
	for _, item := range items {
		fp, ok := item.(*provifv1.PullRequestCommentInfo)
		if !ok {
			logger.Error().Msgf("expected PullRequestCommentInfo, got %T", item)
			continue
		}

		aggregatedCommentBody += fmt.Sprintf("\n\n%s", fp.Body)
	}

	_, err := a.commenter.CommentOnPullRequest(ctx, a.props, provifv1.PullRequestCommentInfo{
		Commit: a.commitSha,
		Body:   aggregatedCommentBody,
	})
	if err != nil {
		return fmt.Errorf("error creating PR review: %w", err)
	}

	return nil
}
