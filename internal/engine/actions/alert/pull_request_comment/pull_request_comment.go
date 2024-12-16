// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package pull_request_comment provides necessary interfaces and implementations for
// processing pull request comment alerts.
package pull_request_comment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
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
	// AlertType is the type of the pull request comment alert engine
	AlertType = "pull_request_comment"
	// PrCommentMaxLength is the maximum length of the pull request comment
	// (this was derived from the limit of the GitHub API)
	PrCommentMaxLength = 65536
)

// Alert is the structure backing the noop alert
type Alert struct {
	actionType interfaces.ActionType
	gh         provifv1.GitHub
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
	Owner      string
	Repo       string
	CommitSha  string
	Number     int
	Comment    string
	Metadata   *alertMetadata
	prevStatus *db.ListRuleEvaluationsByProfileIdRow
}

type alertMetadata struct {
	ReviewID       string     `json:"review_id,omitempty"`
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	PullRequestUrl *string    `json:"pull_request_url,omitempty"`
}

// NewPullRequestCommentAlert creates a new pull request comment alert action
func NewPullRequestCommentAlert(
	actionType interfaces.ActionType,
	reviewCfg *pb.RuleType_Definition_Alert_AlertTypePRComment,
	gh provifv1.GitHub,
	setting models.ActionOpt,
) (*Alert, error) {
	if actionType == "" {
		return nil, fmt.Errorf("action type cannot be empty")
	}

	return &Alert{
		actionType: actionType,
		gh:         gh,
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
		return nil, fmt.Errorf("expected repository, got %T", entity)
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
	logger := zerolog.Ctx(ctx)

	// Process the command
	switch cmd {
	// Create a review
	case interfaces.ActionCmdOn:
		review := &github.PullRequestReviewRequest{
			CommitID: github.String(params.CommitSha),
			Event:    github.String("COMMENT"),
			Body:     github.String(params.Comment),
		}

		r, err := alert.gh.CreateReview(
			ctx,
			params.Owner,
			params.Repo,
			params.Number,
			review,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating PR review: %w, %w", err, enginerr.ErrActionFailed)
		}

		newMeta, err := json.Marshal(alertMetadata{
			ReviewID:       strconv.FormatInt(r.GetID(), 10),
			SubmittedAt:    r.SubmittedAt.GetTime(),
			PullRequestUrl: r.PullRequestURL,
		})
		if err != nil {
			return nil, fmt.Errorf("error marshalling alert metadata json: %w", err)
		}

		logger.Info().Int64("review_id", *r.ID).Msg("PR review created")
		return newMeta, nil
	// Dismiss the review
	case interfaces.ActionCmdOff:
		if params.Metadata == nil {
			// We cannot do anything without the PR review ID, so we assume that turning the alert off is a success
			return nil, fmt.Errorf("no PR comment ID provided: %w", enginerr.ErrActionTurnedOff)
		}

		reviewID, err := strconv.ParseInt(params.Metadata.ReviewID, 10, 64)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Str("review_id", params.Metadata.ReviewID).Msg("failed to parse review_id")
			return nil, fmt.Errorf("no PR comment ID provided: %w, %w", err, enginerr.ErrActionTurnedOff)
		}

		_, err = alert.gh.DismissReview(ctx, params.Owner, params.Repo, params.Number, reviewID,
			&github.PullRequestReviewDismissalRequest{
				Message: github.String("Dismissed due to alert being turned off"),
			})
		if err != nil {
			if errors.Is(err, enginerr.ErrNotFound) {
				// There's no PR review with that ID anymore.
				// We exit by stating that the action was turned off.
				return nil, fmt.Errorf("PR comment already dismissed: %w, %w", err, enginerr.ErrActionTurnedOff)
			}
			return nil, fmt.Errorf("error dismissing PR comment: %w, %w", err, enginerr.ErrActionFailed)
		}
		logger.Info().Str("review_id", params.Metadata.ReviewID).Msg("PR comment dismissed")
		// Success - return ErrActionTurnedOff to indicate the action was successful
		return nil, fmt.Errorf("%s : %w", alert.Class(), enginerr.ErrActionTurnedOff)
	case interfaces.ActionCmdDoNothing:
		// Return the previous alert status.
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
		logger.Info().Msgf("dry run: create a PR comment on PR %d in repo %s/%s with the following body: %s",
			params.Number, params.Owner, params.Repo, *body)
		return nil, nil
	case interfaces.ActionCmdOff:
		if params.Metadata == nil {
			// We cannot do anything without the PR review ID, so we assume that turning the alert off is a success
			return nil, fmt.Errorf("no PR comment ID provided: %w", enginerr.ErrActionTurnedOff)
		}
		logger.Info().Msgf("dry run: dismiss PR comment %s on PR PR %d in repo %s/%s", params.Metadata.ReviewID,
			params.Number, params.Owner, params.Repo)
	case interfaces.ActionCmdDoNothing:
		// Return the previous alert status.
		return alert.runDoNothing(ctx, params)

	}
	return nil, enginerr.ErrActionSkipped
}

// runDoNothing returns the previous alert status
func (_ *Alert) runDoNothing(ctx context.Context, params *paramsPR) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().Str("repo", params.Repo).Logger()

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

// getParamsForSecurityAdvisory extracts the details from the entity
func (alert *Alert) getParamsForPRComment(
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

	// The GitHub Go API takes an int32, but our proto stores an int64; make sure we don't overflow
	// The PR number is an int in GitHub and Gitlab; in practice overflow will never happen.
	if pr.Number > math.MaxInt {
		return nil, fmt.Errorf("pr number is too large")
	}
	result.Number = int(pr.Number)

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
		meta := &alertMetadata{}
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
