// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package issue provides the issue remediation engine.
package issue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	dbadapter "github.com/mindersec/minder/internal/adapters/db"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	enginerr "github.com/mindersec/minder/pkg/engine/errors"
	"github.com/mindersec/minder/pkg/profiles/models"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// RemediateType is the type of the Issue remediation engine.
	RemediateType = "issue"
)

// Limits on the rendered output sent to the repository provider.
const (
	// TitleMaxLength is the maximum number of bytes for the title.
	TitleMaxLength = 75

	// BodyMaxLength is the maximum number of bytes for the body.
	BodyMaxLength = 65536
)

type issueMetadata struct {
	Number int `json:"issue_number,omitempty"`
}

// Remediator implements the issue remediation engine.
type Remediator struct {
	issueCli   provifv1.IssuePublisher
	actionType interfaces.ActionType
	setting    models.ActionOpt

	issueCfg *pb.RuleType_Definition_Remediate_IssueRemediation

	titleTemplate *util.SafeTemplate
	bodyTemplate  *util.SafeTemplate
}

type paramsIssue struct {
	repo       *pb.Repository
	title      string
	body       string
	labels     []string
	assignees  []string
	metadata   *issueMetadata
	prevStatus *db.ListRuleEvaluationsByProfileIdRow
}

// NewIssueRemediate creates a new Issue remediation engine.
func NewIssueRemediate(
	actionType interfaces.ActionType,
	issueCfg *pb.RuleType_Definition_Remediate_IssueRemediation,
	issueCli provifv1.IssuePublisher,
	setting models.ActionOpt,
) (*Remediator, error) {
	err := issueCfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("issue remediation config is invalid: %w", err)
	}

	titleTmpl, err := util.NewSafeHTMLTemplate(&issueCfg.Title, "title")
	if err != nil {
		return nil, fmt.Errorf("cannot parse title template: %w", err)
	}

	bodyTmpl, err := util.NewSafeHTMLTemplate(&issueCfg.Body, "body")
	if err != nil {
		return nil, fmt.Errorf("cannot parse body template: %w", err)
	}

	return &Remediator{
		issueCli: issueCli,
		issueCfg: issueCfg,

		actionType: actionType,
		setting:    setting,

		titleTemplate: titleTmpl,
		bodyTemplate:  bodyTmpl,
	}, nil
}

// TemplateParams is the parameters for the Issue templates
type TemplateParams struct {
	// Entity is the entity being evaluated.
	Entity any
	// Profile contains the profile definition.
	Profile map[string]any
	// Params contains the rule instance parameters.
	Params map[string]any
	// EvalResultOutput contains the evaluation output.
	EvalResultOutput any
}

// Class returns the action type of the remediation engine.
func (r *Remediator) Class() interfaces.ActionType {
	return r.actionType
}

// Type returns the action subtype of the remediation engine.
func (*Remediator) Type() string {
	return RemediateType
}

// GetOnOffState returns the remediation action state read from the profile.
func (r *Remediator) GetOnOffState() models.ActionOpt {
	return models.ActionOptOrDefault(r.setting, models.ActionOptOff)
}

func (r *Remediator) run(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	p *paramsIssue,
) (json.RawMessage, error) {
	switch cmd {
	case interfaces.ActionCmdOn:
		return r.runOn(ctx, p)

	case interfaces.ActionCmdOff:
		return r.runOff(ctx, p)

	case interfaces.ActionCmdDoNothing:
		return r.runDoNothing(ctx, p)
	}

	return nil, fmt.Errorf("unimplemented action command: %v", cmd)
}

// Do perform the remediation
func (r *Remediator) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	ent proto.Message,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (json.RawMessage, error) {
	p, err := r.getParamsForIssueRemediation(ctx, ent, params, metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot get issue remediation params: %w", err)
	}

	switch r.setting {
	case models.ActionOptOn:
		return r.run(ctx, cmd, p)

	case models.ActionOptDryRun:
		return r.dryRun(ctx, cmd, p)

	case models.ActionOptOff, models.ActionOptUnknown:
		return nil, errors.New("unexpected remediation action setting")

	default:
		return nil, fmt.Errorf("unexpected remediation action setting: %v", r.setting)
	}
}

func (r *Remediator) getParamsForIssueRemediation(
	ctx context.Context,
	ent proto.Message,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (*paramsIssue, error) {

	repo, ok := ent.(*pb.Repository)
	if !ok {
		return nil, fmt.Errorf("expected repository, got %T", ent)
	}

	tmplParams := &TemplateParams{
		Entity:  ent,
		Profile: params.GetRule().Def,
		Params:  params.GetRule().Params,
	}

	if params.GetEvalResult() != nil {
		tmplParams.EvalResultOutput = params.GetEvalResult().Output
	}

	title, err := r.titleTemplate.Render(ctx, tmplParams, TitleMaxLength)
	if err != nil {
		return nil, fmt.Errorf("cannot render title template: %w", err)
	}

	body, err := r.bodyTemplate.Render(ctx, tmplParams, BodyMaxLength)
	if err != nil {
		return nil, fmt.Errorf("cannot render body template: %w", err)
	}

	// Unmarshal existing remediation metadata, if any.
	meta := &issueMetadata{}
	if metadata != nil {
		if err := json.Unmarshal(*metadata, meta); err != nil {
			// Metadata may not exist yet; log and continue.
			return nil, fmt.Errorf("cannot unmarshal remediation metadata: %w", err)
		}
	}

	labels := r.issueCfg.GetLabels()
	if labels == nil {
		labels = []string{}
	}
	assignees := r.issueCfg.GetAssignees()
	if assignees == nil {
		assignees = []string{}
	}

	return &paramsIssue{
		repo:       repo,
		title:      title,
		body:       body,
		labels:     labels,
		assignees:  assignees,
		metadata:   meta,
		prevStatus: params.GetEvalStatusFromDb(),
	}, nil
}

func (r *Remediator) dryRun(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	p *paramsIssue,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("repo", p.repo.String()).
		Logger()

	// Process the command
	switch cmd {
	case interfaces.ActionCmdOn:
		logger.Info().
			Str("title", p.title).
			Str("body", p.body).
			Strs("labels", p.labels).
			Strs("assignees", p.assignees).
			Msg("issue remediation dry run")

		return nil, nil

	case interfaces.ActionCmdOff:
		if p.metadata == nil || p.metadata.Number == 0 {
			// We cannot do anything without an issue number, so we assume that closing this is a success.
			return nil, fmt.Errorf("no issue number provided: %w", enginerr.ErrActionSkipped)
		}

		logger.Info().
			Int("issue_number", p.metadata.Number).
			Str("owner", p.repo.GetOwner()).
			Str("repo", p.repo.GetName()).
			Msg("would close issue")

		return nil, nil

	case interfaces.ActionCmdDoNothing:
		return r.runDoNothing(ctx, p)
	}

	return nil, nil
}

func (r *Remediator) runOn(
	ctx context.Context,
	p *paramsIssue,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("repo", p.repo.String()).
		Logger()

	// If we already have an issue recorded in the remediation metadata,
	// don't create another one.
	if p.metadata != nil && p.metadata.Number != 0 {
		logger.Debug().
			Int("issue_number", p.metadata.Number).
			Msg("issue already exists")

		newMeta, err := json.Marshal(*p.metadata)
		if err != nil {
			return nil, fmt.Errorf("error marshalling issue remediation metadata json: %w", err)
		}

		return newMeta, enginerr.ErrActionPending
	}

	issue, err := r.issueCli.CreateIssue(
		ctx,
		p.repo.GetOwner(),
		p.repo.GetName(),
		p.title,
		p.body,
		p.labels,
		p.assignees,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot create issue: %w, %w",
			err,
			enginerr.ErrActionFailed,
		)
	}

	newMeta, err := json.Marshal(issueMetadata{
		Number: issue.GetNumber(),
	})
	if err != nil {
		return nil, fmt.Errorf(
			"error marshalling issue remediation metadata json: %w",
			err,
		)
	}

	logger.Info().
		Int("issue_number", issue.GetNumber()).
		Msg("issue created")

	return newMeta, enginerr.ErrActionPending
}

func (r *Remediator) runOff(
	ctx context.Context,
	p *paramsIssue,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("repo", p.repo.String()).
		Logger()

	if p.metadata == nil || p.metadata.Number == 0 {
		// We cannot do anything without an issue number, so we assume that closing this is a success.
		return nil, fmt.Errorf("no issue number provided: %w", enginerr.ErrActionSkipped)
	}

	issue, err := r.issueCli.CloseIssue(
		ctx,
		p.repo.GetOwner(),
		p.repo.GetName(),
		p.metadata.Number,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error closing issue %d: %w, %w",
			p.metadata.Number,
			err,
			enginerr.ErrActionFailed,
		)
	}

	logger.Info().
		Int("issue_number", issue.GetNumber()).
		Msg("issue closed")

	return nil, enginerr.ErrActionSkipped
}

// runDoNothing returns the previous remediation status.
func (*Remediator) runDoNothing(
	ctx context.Context,
	p *paramsIssue,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("repo", p.repo.String()).
		Logger()

	logger.Debug().Msg("Running do nothing")

	err := dbadapter.RemediationStatusAsError(p.prevStatus)

	if p.prevStatus != nil {
		return p.prevStatus.RemMetadata, err
	}

	return nil, err
}
