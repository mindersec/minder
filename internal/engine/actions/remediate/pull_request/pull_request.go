// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package pull_request provides the pull request remediation engine
package pull_request

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/db"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	engifv1 "github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// RemediateType is the type of the REST remediation engine
	RemediateType = "pull_request"
)

const (
	// if no Mode is specified, create a regular file with 0644 UNIX permissions
	ghModeNonExecFile = "100644"
	dflBranchBaseName = "minder"
)

const (
	prTemplateName = "prBody"
	prBodyTmplStr  = "{{.PrText}}"
)

const (
	// TitleMaxLength is the maximum number of bytes for the title
	TitleMaxLength = 256

	// BodyMaxLength is the maximum number of bytes for the body
	BodyMaxLength = 5120
)

type pullRequestMetadata struct {
	Number int `json:"pr_number,omitempty"`
}

// Remediator is the remediation engine for the Pull Request remediation type
type Remediator struct {
	ghCli      provifv1.GitHub
	actionType interfaces.ActionType
	setting    models.ActionOpt

	prCfg                *pb.RuleType_Definition_Remediate_PullRequestRemediation
	modificationRegistry modificationRegistry

	titleTemplate *util.SafeTemplate
	bodyTemplate  *util.SafeTemplate
}

type paramsPR struct {
	ingested   *engifv1.Ingested
	repo       *pb.Repository
	title      string
	modifier   fsModifier
	body       string
	metadata   *pullRequestMetadata
	prevStatus *db.ListRuleEvaluationsByProfileIdRow
}

// NewPullRequestRemediate creates a new PR remediation engine
func NewPullRequestRemediate(
	actionType interfaces.ActionType,
	prCfg *pb.RuleType_Definition_Remediate_PullRequestRemediation,
	ghCli provifv1.GitHub,
	setting models.ActionOpt,
) (*Remediator, error) {
	err := prCfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("pull request remediation config is invalid: %w", err)
	}

	titleTmpl, err := util.NewSafeHTMLTemplate(&prCfg.Title, "title")
	if err != nil {
		return nil, fmt.Errorf("cannot parse title template: %w", err)
	}

	bodyTmpl, err := util.NewSafeHTMLTemplate(&prCfg.Body, "body")
	if err != nil {
		return nil, fmt.Errorf("cannot parse body template: %w", err)
	}

	modRegistry := newModificationRegistry()
	modRegistry.registerBuiltIn()

	return &Remediator{
		ghCli:                ghCli,
		prCfg:                prCfg,
		actionType:           actionType,
		modificationRegistry: modRegistry,
		setting:              setting,

		titleTemplate: titleTmpl,
		bodyTemplate:  bodyTmpl,
	}, nil
}

// PrTemplateParams is the parameters for the PR templates
type PrTemplateParams struct {
	// Entity is the entity to be evaluated
	Entity any
	// Profile are the parameters to be used in the template
	Profile map[string]any
	// Params are the rule instance parameters
	Params map[string]any
	// EvalResultOutput is the data output by the rule evaluation engine
	EvalResultOutput any
}

// Class returns the action type of the remediation engine
func (r *Remediator) Class() interfaces.ActionType {
	return r.actionType
}

// Type returns the action subtype of the remediation engine
func (*Remediator) Type() string {
	return RemediateType
}

// GetOnOffState returns the alert action state read from the profile
func (r *Remediator) GetOnOffState() models.ActionOpt {
	return models.ActionOptOrDefault(r.setting, models.ActionOptOff)
}

// Do perform the remediation
func (r *Remediator) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	ent proto.Message,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (json.RawMessage, error) {
	p, err := r.getParamsForPRRemediation(ctx, ent, params, metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot get PR remediation params: %w", err)
	}
	var remErr error
	switch r.setting {
	case models.ActionOptOn:
		return r.run(ctx, cmd, p)
	case models.ActionOptDryRun:
		return r.dryRun(ctx, cmd, p)
	case models.ActionOptOff, models.ActionOptUnknown:
		remErr = errors.New("unexpected action")
	}
	return nil, remErr
}

func (r *Remediator) getParamsForPRRemediation(
	ctx context.Context,
	ent proto.Message,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (*paramsPR, error) {
	logger := zerolog.Ctx(ctx)

	repo, ok := ent.(*pb.Repository)
	if !ok {
		return nil, fmt.Errorf("expected repository, got %T", ent)
	}

	tmplParams := &PrTemplateParams{
		Entity:  ent,
		Profile: params.GetRule().Def,
		Params:  params.GetRule().Params,
	}

	if params.GetEvalResult() != nil {
		tmplParams.EvalResultOutput = params.GetEvalResult().Output
	}

	ingested := params.GetIngestResult()
	if ingested == nil || ingested.Fs == nil || ingested.Storer == nil {
		return nil, errors.New("ingested filesystem is nil or no git repo was ingested")
	}

	title, err := r.titleTemplate.Render(ctx, tmplParams, TitleMaxLength)
	if err != nil {
		return nil, fmt.Errorf("cannot execute title template: %w", err)
	}

	modification, err := r.modificationRegistry.getModification(getMethod(r.prCfg), &modificationConstructorParams{
		prCfg: r.prCfg,
		ghCli: r.ghCli,
		bfs:   ingested.Fs,
		def:   params.GetRule().Def,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get modification: %w", err)
	}

	err = modification.createFsModEntries(ctx, ent, params)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR entries: %w", err)
	}

	prFullBodyText, err := r.getPrBodyText(ctx, tmplParams)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR full body text: %w", err)
	}

	// Unmarshal the existing remediation metadata, if any
	meta := &pullRequestMetadata{}
	if metadata != nil {
		err := json.Unmarshal(*metadata, meta)
		if err != nil {
			// There's nothing saved apparently, so no need to fail here, but do log the error
			logger.Debug().Msgf("error unmarshalling remediation metadata: %v", err)
		}
	}
	return &paramsPR{
		ingested:   ingested,
		repo:       repo,
		title:      title,
		modifier:   modification,
		body:       prFullBodyText,
		metadata:   meta,
		prevStatus: params.GetEvalStatusFromDb(),
	}, nil
}

func (r *Remediator) dryRun(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	p *paramsPR,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).Info().Str("repo", p.repo.String())
	// Process the command
	switch cmd {
	case interfaces.ActionCmdOn:
		// TODO: jsonize too
		logger.Msgf("title:\n%s\n", p.title)
		logger.Msgf("body:\n%s\n", p.body)

		err := p.modifier.writeSummary(os.Stdout)
		if err != nil {
			logger.Msgf("cannot write summary: %s\n", err)
		}
		return nil, nil
	case interfaces.ActionCmdOff:
		if p.metadata == nil || p.metadata.Number == 0 {
			// We cannot do anything without a PR number, so we assume that closing this is a success
			return nil, fmt.Errorf("no pull request number provided: %w", enginerr.ErrActionSkipped)
		}
		endpoint := fmt.Sprintf("repos/%v/%v/pulls/%d", p.repo.GetOwner(), p.repo.GetName(), p.metadata.Number)
		body := "{\"state\": \"closed\"}"
		curlCmd, err := util.GenerateCurlCommand(ctx, "PATCH", r.ghCli.GetBaseURL(), endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("cannot generate curl command to close a pull request: %w", err)
		}
		logger.Msgf("run the following curl command: \n%s\n", curlCmd)
		return nil, nil
	case interfaces.ActionCmdDoNothing:
		return r.runDoNothing(ctx, p)
	}
	return nil, nil
}
func (r *Remediator) runOn(
	ctx context.Context,
	p *paramsPR,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().Str("repo", p.repo.String()).Logger()
	repo, err := git.Open(p.ingested.Storer, p.ingested.Fs)
	if err != nil {
		return nil, fmt.Errorf("cannot open git repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("cannot get worktree: %w", err)
	}

	logger.Debug().Msg("Getting authenticated user details")
	email, err := r.ghCli.GetPrimaryEmail(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get primary email: %w", err)
	}

	currentHeadReference, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("cannot get current HEAD: %w", err)
	}
	currHeadName := currentHeadReference.Name()

	// This resets the worktree so we don't corrupt the ingest cache (at least the main/originally-fetched branch).
	// This also makes sure, all new remediations check out from main branch rather than prev remediation branch.
	defer checkoutToOriginallyFetchedBranch(&logger, wt, currHeadName)

	logger.Debug().Str("branch", branchBaseName(p.title)).Msg("Checking out branch")
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchBaseName(p.title)),
		Create: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot checkout branch: %w", err)
	}

	logger.Debug().Msg("Creating file entries")
	changeEntries, err := p.modifier.modifyFs()
	if err != nil {
		return nil, fmt.Errorf("cannot modifyFs: %w", err)
	}

	logger.Debug().Msg("Staging changes")
	for _, entry := range changeEntries {
		if _, err := wt.Add(entry.Path); err != nil {
			return nil, fmt.Errorf("cannot add file %s: %w", entry.Path, err)
		}
	}

	logger.Debug().Msg("Committing changes")
	_, err = wt.Commit(p.title, &git.CommitOptions{
		Author: &object.Signature{
			Name:  userNameForCommit(ctx, r.ghCli),
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot commit: %w", err)
	}

	refspec := refFromBranch(branchBaseName(p.title))

	l := logger.With().Str("branchBaseName", branchBaseName(p.title)).Logger()

	// Check if a PR already exists for this branch
	prNumber := getPRNumberFromBranch(ctx, r.ghCli, p.repo, branchBaseName(p.title))

	// If no PR exists, push the branch and create a PR
	if prNumber == 0 {
		err = pushBranch(ctx, repo, refspec, r.ghCli)
		if err != nil {
			return nil, fmt.Errorf("cannot push branch: %w", err)
		}

		pr, err := r.ghCli.CreatePullRequest(
			ctx, p.repo.GetOwner(), p.repo.GetName(),
			p.title, p.body,
			refspec,
			currHeadName.Short(),
		)
		if err != nil {
			return nil, fmt.Errorf("cannot create pull request: %w, %w", err, enginerr.ErrActionFailed)
		}
		// Return the new PR number
		prNumber = pr.GetNumber()
		l = l.With().Str("pr_origin", "newly_created").Logger()
	} else {
		l = l.With().Str("pr_origin", "already_existed").Logger()
	}

	newMeta, err := json.Marshal(pullRequestMetadata{Number: prNumber})
	if err != nil {
		return nil, fmt.Errorf("error marshalling pull request remediation metadata json: %w", err)
	}
	// Success - return the new metadata for storing the pull request number
	l.Info().Int("pr_number", prNumber).Msg("pull request remediation completed")
	return newMeta, enginerr.ErrActionPending
}

func getPRNumberFromBranch(
	ctx context.Context,
	cli provifv1.GitHub,
	repo *pb.Repository,
	branchName string,
) int {
	opts := &github.PullRequestListOptions{
		// TODO: This is not working as expected, need to fix this
		// Head: fmt.Sprintf("%s:%s", repo.GetOwner(), branchName),
		State: "open",
	}
	openPrs, err := cli.ListPullRequests(ctx, repo.GetOwner(), repo.GetName(), opts)
	if err != nil {
		return 0
	}
	for _, pr := range openPrs {
		if pr.GetHead().GetRef() == branchName {
			return pr.GetNumber()
		}
	}
	return 0
}

func (r *Remediator) runOff(
	ctx context.Context,
	p *paramsPR,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().Str("repo", p.repo.String()).Logger()

	if p.metadata == nil || p.metadata.Number == 0 {
		// We cannot do anything without a PR number, so we assume that closing this is a success
		return nil, fmt.Errorf("no pull request number provided: %w", enginerr.ErrActionSkipped)
	}

	pr, err := r.ghCli.ClosePullRequest(ctx, p.repo.GetOwner(), p.repo.GetName(), p.metadata.Number)
	if err != nil {
		return nil, fmt.Errorf("error closing pull request %d: %w, %w", p.metadata.Number, err, enginerr.ErrActionFailed)
	}
	logger.Info().Int("pr_number", pr.GetNumber()).Msg("pull request closed")
	return nil, enginerr.ErrActionSkipped
}

func (r *Remediator) run(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	p *paramsPR,
) (json.RawMessage, error) {
	// Process the command
	switch cmd {
	case interfaces.ActionCmdOn:
		return r.runOn(ctx, p)
	case interfaces.ActionCmdOff:
		return r.runOff(ctx, p)
	case interfaces.ActionCmdDoNothing:
		return r.runDoNothing(ctx, p)
	}
	return nil, enginerr.ErrActionSkipped
}

func pushBranch(ctx context.Context, repo *git.Repository, refspec string, gh provifv1.GitHub) error {
	var b bytes.Buffer
	pushOptions := &git.PushOptions{
		RemoteName: guessRemote(repo),
		Force:      true,
		RefSpecs: []config.RefSpec{
			config.RefSpec(
				fmt.Sprintf("+%s:%s", refspec, refspec),
			),
		},
		Progress: &b,
	}
	err := gh.AddAuthToPushOptions(ctx, pushOptions)
	if err != nil {
		return fmt.Errorf("cannot add auth to push options: %w", err)
	}

	err = repo.PushContext(ctx, pushOptions)
	if err != nil {
		return fmt.Errorf("cannot push: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msgf("Push output: %s", b.String())
	return nil
}

func guessRemote(gitRepo *git.Repository) string {
	remotes, err := gitRepo.Remotes()
	if err != nil {
		return ""
	}

	if len(remotes) == 0 {
		return ""
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			return remote.Config().Name
		}
	}

	return remotes[0].Config().Name
}

func refFromBranch(branchFrom string) string {
	return fmt.Sprintf("refs/heads/%s", branchFrom)
}

func branchBaseName(prTitle string) string {
	baseName := dflBranchBaseName
	normalizedPrTitle := strings.ReplaceAll(strings.ToLower(prTitle), " ", "_")
	return fmt.Sprintf("%s_%s", baseName, normalizedPrTitle)
}

func userNameForCommit(ctx context.Context, gh provifv1.GitHub) string {
	var name string

	// we ignore errors here, as we can still create a commit without a name
	// and errors are checked when getting the primary email
	name, _ = gh.GetName(ctx)
	if name == "" {
		name, _ = gh.GetLogin(ctx)
	}
	return name
}

func (r *Remediator) getPrBodyText(ctx context.Context, tmplParams *PrTemplateParams) (string, error) {
	body := new(bytes.Buffer)
	if err := r.bodyTemplate.Execute(ctx, body, tmplParams, BodyMaxLength); err != nil {
		return "", fmt.Errorf("cannot execute body template: %w", err)
	}

	prFullBodyText, err := createReviewBody(body.String())
	if err != nil {
		return "", fmt.Errorf("cannot create PR full body text: %w", err)
	}

	return prFullBodyText, nil
}

func getMethod(prCfg *pb.RuleType_Definition_Remediate_PullRequestRemediation) string {
	if prCfg.Method == "" {
		return minderContentModification
	}

	return prCfg.Method
}

func createReviewBody(prText string) (string, error) {
	tmpl, err := template.New(prTemplateName).Option("missingkey=error").Parse(prBodyTmplStr)
	if err != nil {
		return "", err
	}

	data := struct {
		PrText string
	}{
		PrText: prText,
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func checkoutToOriginallyFetchedBranch(
	logger *zerolog.Logger,
	wt *git.Worktree,
	originallyFetchedBranch plumbing.ReferenceName,
) {
	err := wt.Checkout(&git.CheckoutOptions{
		Branch: originallyFetchedBranch,
	})
	if err != nil {
		logger.Err(err).Msg(
			"unable to checkout to the previous head, this can corrupt the ingest cache, should not happen",
		)
	} else {
		logger.Info().Msg(fmt.Sprintf("checked out back to %s branch", originallyFetchedBranch))
	}
}

// runDoNothing returns the previous remediation status
func (*Remediator) runDoNothing(ctx context.Context, p *paramsPR) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().Str("repo", p.repo.String()).Logger()

	logger.Debug().Msg("Running do nothing")

	// Return the previous remediation status.
	err := enginerr.RemediationStatusAsError(p.prevStatus)
	// If there is a valid remediation metadata, return it too
	if p.prevStatus != nil {
		return p.prevStatus.RemMetadata, err
	}
	// If there is no remediation metadata, return nil as the metadata and the error
	return nil, err
}
