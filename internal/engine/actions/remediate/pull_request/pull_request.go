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

// Package pull_request provides the pull request remediation engine
package pull_request

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	enginerr "github.com/stacklok/minder/internal/engine/errors"
	htmltemplate "html/template"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// RemediateType is the type of the REST remediation engine
	RemediateType = "pull_request"
)

const (
	// if no Mode is specified, create a regular file with 0644 UNIX permissions
	ghModeNonExecFile = "100644"
	dflBranchBaseName = "minder"
	dflBranchTo       = "main"
)

const (
	prTemplateName = "prBody"
	prBodyTmplStr  = "{{.PrText}}"
)

type pullRequestMetadata struct {
	Number int `json:"pr_number,omitempty"`
}

// Remediator is the remediation engine for the Pull Request remediation type
type Remediator struct {
	ghCli      provifv1.GitHub
	actionType interfaces.ActionType

	prCfg                *pb.RuleType_Definition_Remediate_PullRequestRemediation
	modificationRegistry modificationRegistry

	titleTemplate *htmltemplate.Template
	bodyTemplate  *htmltemplate.Template
}

// NewPullRequestRemediate creates a new PR remediation engine
func NewPullRequestRemediate(
	actionType interfaces.ActionType,
	prCfg *pb.RuleType_Definition_Remediate_PullRequestRemediation,
	pbuild *providers.ProviderBuilder,
) (*Remediator, error) {
	err := prCfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("pull request remediation config is invalid: %w", err)
	}

	titleTmpl, err := util.ParseNewHtmlTemplate(&prCfg.Title, "title")
	if err != nil {
		return nil, fmt.Errorf("cannot parse title template: %w", err)
	}

	bodyTmpl, err := util.ParseNewHtmlTemplate(&prCfg.Body, "body")
	if err != nil {
		return nil, fmt.Errorf("cannot parse body template: %w", err)
	}

	ghCli, err := pbuild.GetGitHub()
	if err != nil {
		return nil, fmt.Errorf("failed to get github client: %w", err)
	}

	modRegistry := newModificationRegistry()
	modRegistry.registerBuiltIn()

	return &Remediator{
		ghCli:                ghCli,
		prCfg:                prCfg,
		actionType:           actionType,
		modificationRegistry: modRegistry,

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
}

// Class returns the action type of the remediation engine
func (r *Remediator) Class() interfaces.ActionType {
	return r.actionType
}

// Type returns the action subtype of the remediation engine
func (_ *Remediator) Type() string {
	return RemediateType
}

// GetOnOffState returns the alert action state read from the profile
func (_ *Remediator) GetOnOffState(p *pb.Profile) interfaces.ActionOpt {
	return interfaces.ActionOptFromString(p.Remediate, interfaces.ActionOptOff)
}

type paramsPR struct {
	ingested *interfaces.Result
	repo     *pb.Repository
	title    string
	modifier fsModifier
	body     string
	metadata *pullRequestMetadata
}

// Do performs the remediation
func (r *Remediator) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	setting interfaces.ActionOpt,
	ent protoreflect.ProtoMessage,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (json.RawMessage, error) {
	p, err := r.getParamsForPRRemediation(ctx, ent, params, metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot get PR remediation params: %w", err)
	}
	var remErr error
	switch setting {
	case interfaces.ActionOptOn:
		return r.run(ctx, cmd, p)
	case interfaces.ActionOptDryRun:
		r.dryRun(p)
		remErr = nil
	case interfaces.ActionOptOff, interfaces.ActionOptUnknown:
		remErr = errors.New("unexpected action")
	}
	return nil, remErr
}

func (r *Remediator) getParamsForPRRemediation(ctx context.Context, ent protoreflect.ProtoMessage, params interfaces.ActionsParams, metadata *json.RawMessage) (*paramsPR, error) {
	logger := zerolog.Ctx(ctx)

	repo, ok := ent.(*pb.Repository)
	if !ok {
		return nil, fmt.Errorf("expected repository, got %T", ent)
	}

	tmplParams := &PrTemplateParams{
		Entity:  ent,
		Profile: params.GetRule().Def.AsMap(),
		Params:  params.GetRule().Params.AsMap(),
	}

	ingested := params.GetIngestResult()
	if ingested == nil || ingested.Fs == nil || ingested.Storer == nil {
		return nil, errors.New("ingested filesystem is nil or no git repo was ingested")
	}

	title := new(bytes.Buffer)
	if err := r.titleTemplate.Execute(title, tmplParams); err != nil {
		return nil, fmt.Errorf("cannot execute title template: %w", err)
	}

	modification, err := r.modificationRegistry.getModification(getMethod(r.prCfg), &modificationConstructorParams{
		prCfg: r.prCfg,
		ghCli: r.ghCli,
		bfs:   ingested.Fs,
		def:   params.GetRule().Def.AsMap(),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get modification: %w", err)
	}

	err = modification.createFsModEntries(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR entries: %w", err)
	}

	prFullBodyText, err := r.getPrBodyText(tmplParams)
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
		} else {
		}
	}

	return &paramsPR{
		ingested: ingested,
		repo:     repo,
		title:    title.String(),
		modifier: modification,
		body:     prFullBodyText,
		metadata: meta,
	}, nil
}

func (_ *Remediator) dryRun(p *paramsPR) {
	// TODO: jsonize too
	fmt.Printf("title:\n%s\n", p.title)
	fmt.Printf("body:\n%s\n", p.body)

	err := p.modifier.writeSummary(os.Stdout)
	if err != nil {
		fmt.Printf("cannot write summary: %s\n", err)
	}
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

	err = pushBranch(ctx, repo, refspec, r.ghCli)
	if err != nil {
		return nil, fmt.Errorf("cannot push branch: %w", err)
	}

	pr, err := r.ghCli.CreatePullRequest(
		ctx, p.repo.GetOwner(), p.repo.GetName(),
		p.title, p.body,
		refspec,
		dflBranchTo,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create pull request: %w", err)
	}
	newMeta, err := json.Marshal(pullRequestMetadata{Number: pr.GetNumber()})
	if err != nil {
		return nil, fmt.Errorf("error marshalling pull request remediation metadata json: %w", err)
	}
	// Success - return the new metadata for storing the pull request number
	logger.Info().Int("pr_number", pr.GetNumber()).Msg("pull request created")
	return newMeta, enginerr.ErrActionPending
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

	err := r.ghCli.ClosePullRequest(ctx, p.repo.GetOwner(), p.repo.GetName(), strconv.Itoa(p.metadata.Number))
	if err != nil {
		if errors.Is(err, enginerr.ErrNotFound) {
			// There's no pull request with such PR number anymore (perhaps it was closed manually).
			// We exit by stating that the action was turned off.
			return nil, fmt.Errorf("pull request already closed: %w, %w", err, enginerr.ErrActionSkipped)
		}
		return nil, fmt.Errorf("error closing pull request: %w, %w", err, enginerr.ErrActionFailed)
	}
	logger.Info().Int("pr_number", p.metadata.Number).Msg("pull request closed")
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
		return nil, enginerr.ErrActionSkipped
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

func (r *Remediator) getPrBodyText(tmplParams *PrTemplateParams) (string, error) {
	body := new(bytes.Buffer)
	if err := r.bodyTemplate.Execute(body, tmplParams); err != nil {
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
