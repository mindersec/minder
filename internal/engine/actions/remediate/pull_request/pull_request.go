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
	htmltemplate "html/template"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	"github.com/google/go-github/v56/github"
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
	prMagicTemplateName = "prMagicComment"
	prBodyMagicTemplate = `<!-- minder: pr-remediation-body: { "ContentSha": "{{.ContentSha}}" } -->`

	prTemplateName = "prBody"
	prBodyTmplStr  = "{{.MagicComment}}\n\n{{.PrText}}"
)

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

// Do performs the remediation
func (r *Remediator) Do(
	ctx context.Context,
	_ interfaces.ActionCmd,
	remAction interfaces.ActionOpt,
	ent protoreflect.ProtoMessage,
	params interfaces.ActionsParams,
	_ *json.RawMessage,
) (json.RawMessage, error) {
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

	magicComment, err := r.prMagicComment(modification)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR magic comment: %w", err)
	}

	prFullBodyText, err := r.getPrBodyText(tmplParams, magicComment)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR full body text: %w", err)
	}

	var remErr error
	switch remAction {
	case interfaces.ActionOptOn:
		alreadyExists, err := prWithContentAlreadyExists(ctx, r.ghCli, repo, magicComment)
		if err != nil {
			return nil, fmt.Errorf("cannot check if PR already exists: %w", err)
		}
		if alreadyExists {
			zerolog.Ctx(ctx).Info().Msg("PR already exists, won't create a new one")
			return nil, nil
		}
		remErr = r.runGit(ctx, ingested.Fs, ingested.Storer, modification, repo, title.String(), prFullBodyText)
	case interfaces.ActionOptDryRun:
		r.dryRun(modification, title.String(), prFullBodyText)
		remErr = nil
	case interfaces.ActionOptOff, interfaces.ActionOptUnknown:
		remErr = errors.New("unexpected action")
	}
	return nil, remErr
}

func (_ *Remediator) dryRun(modifier fsModifier, title, body string) {
	// TODO: jsonize too
	fmt.Printf("title:\n%s\n", title)
	fmt.Printf("body:\n%s\n", body)

	err := modifier.writeSummary(os.Stdout)
	if err != nil {
		fmt.Printf("cannot write summary: %s\n", err)
	}
}

func (r *Remediator) runGit(
	ctx context.Context,
	fs billy.Filesystem,
	storer storage.Storer,
	modifier fsModifier,
	pbRepo *pb.Repository,
	title, body string,
) error {
	logger := zerolog.Ctx(ctx).With().Str("repo", pbRepo.String()).Logger()

	repo, err := git.Open(storer, fs)
	if err != nil {
		return fmt.Errorf("cannot open git repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("cannot get worktree: %w", err)
	}

	logger.Debug().Msg("Getting authenticated user details")
	username, err := r.ghCli.GetUsername(ctx)
	if err != nil {
		return fmt.Errorf("cannot get username: %w", err)
	}

	email, err := r.ghCli.GetPrimaryEmail(ctx)
	if err != nil {
		return fmt.Errorf("cannot get primary email: %w", err)
	}

	currentHeadReference, err := repo.Head()
	if err != nil {
		return fmt.Errorf("cannot get current HEAD: %w", err)
	}
	currHeadName := currentHeadReference.Name()

	// This resets the worktree so we don't corrupt the ingest cache (at least the main/originally-fetched branch).
	// This also makes sure, all new remediations check out from main branch rather than prev remediation branch.
	defer checkoutToOriginallyFetchedBranch(&logger, wt, currHeadName)

	logger.Debug().Str("branch", branchBaseName(title)).Msg("Checking out branch")
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchBaseName(title)),
		Create: true,
	})
	if err != nil {
		return fmt.Errorf("cannot checkout branch: %w", err)
	}

	logger.Debug().Msg("Creating file entries")
	changeEntries, err := modifier.modifyFs()
	if err != nil {
		return fmt.Errorf("cannot modifyFs: %w", err)
	}

	logger.Debug().Msg("Staging changes")
	for _, entry := range changeEntries {
		if _, err := wt.Add(entry.Path); err != nil {
			return fmt.Errorf("cannot add file %s: %w", entry.Path, err)
		}
	}

	logger.Debug().Msg("Committing changes")
	_, err = wt.Commit(title, &git.CommitOptions{
		Author: &object.Signature{
			Name:  username,
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("cannot commit: %w", err)
	}

	refspec := refFromBranch(branchBaseName(title))

	err = pushBranch(ctx, repo, refspec, r.ghCli)
	if err != nil {
		return fmt.Errorf("cannot push branch: %w", err)
	}

	// if a PR from this branch already exists, don't create a new one
	// this handles the case where the content changed (e.g. profile changed)
	// but the PR was not closed
	prAlreadyExists, err := prFromBranchAlreadyExists(ctx, r.ghCli, pbRepo, branchBaseName(title))
	if err != nil {
		return fmt.Errorf("cannot check if PR from branch already exists: %w", err)
	}

	if prAlreadyExists {
		zerolog.Ctx(ctx).Info().Msg("PR from branch already exists, won't create a new one")
		return nil
	}

	_, err = r.ghCli.CreatePullRequest(
		ctx, pbRepo.GetOwner(), pbRepo.GetName(),
		title, body,
		refspec,
		dflBranchTo,
	)
	if err != nil {
		return fmt.Errorf("cannot create pull request: %w", err)
	}

	zerolog.Ctx(ctx).Info().Msg("Pull request created")
	return nil
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

func (_ *Remediator) prMagicComment(modifier fsModifier) (string, error) {
	tmpl, err := template.New(prMagicTemplateName).Option("missingkey=error").Parse(prBodyMagicTemplate)
	if err != nil {
		return "", err
	}

	contentSha, err := modifier.hash()
	if err != nil {
		return "", fmt.Errorf("cannot get content sha1: %w", err)
	}

	data := struct {
		ContentSha string
	}{
		ContentSha: contentSha,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r *Remediator) getPrBodyText(tmplParams *PrTemplateParams, magicComment string) (string, error) {
	body := new(bytes.Buffer)
	if err := r.bodyTemplate.Execute(body, tmplParams); err != nil {
		return "", fmt.Errorf("cannot execute body template: %w", err)
	}

	prFullBodyText, err := createReviewBody(body.String(), magicComment)
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

func createReviewBody(prText, magicComment string) (string, error) {
	tmpl, err := template.New(prTemplateName).Option("missingkey=error").Parse(prBodyTmplStr)
	if err != nil {
		return "", err
	}

	data := struct {
		MagicComment string
		PrText       string
	}{
		MagicComment: magicComment,
		PrText:       prText,
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// returns true if an open PR with the magic comment already exists
func prWithContentAlreadyExists(
	ctx context.Context,
	cli provifv1.GitHub,
	repo *pb.Repository,
	magicComment string,
) (bool, error) {
	openPrs, err := cli.ListPullRequests(ctx, repo.GetOwner(), repo.GetName(), &github.PullRequestListOptions{})
	if err != nil {
		return false, fmt.Errorf("cannot list pull requests: %w", err)
	}

	for _, pr := range openPrs {
		if strings.Contains(pr.GetBody(), magicComment) {
			return true, nil
		}
	}
	return false, nil
}

func prFromBranchAlreadyExists(
	ctx context.Context,
	cli provifv1.GitHub,
	repo *pb.Repository,
	branchName string,
) (bool, error) {
	// TODO(jakub): pagination
	opts := &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:%s", repo.GetOwner(), branchName),
	}

	openPrs, err := cli.ListPullRequests(ctx, repo.GetOwner(), repo.GetName(), opts)
	if err != nil {
		return false, fmt.Errorf("cannot list pull requests: %w", err)
	}

	return len(openPrs) > 0, nil
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
