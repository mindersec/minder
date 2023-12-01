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
	"crypto/sha1" // #nosec G505 - we're not using sha1 for crypto, only to quickly compare contents
	"encoding/json"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
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
	dflBranchFrom     = "main"
	dflBranchTo       = "main"
)

const (
	prMagicTemplateName = "prMagicComment"
	prBodyMagicTemplate = `<!-- minder: pr-remediation-body: { "ContentSha": "{{.ContentSha}}" } -->`

	prTemplateName = "prBody"
	prBodyTmplStr  = "{{.MagicComment}}\n\n{{.PrText}}"

	dryRunTemplateName = "dryRun"
	dryRunTmpl         = `{{- range . }}
Path: {{ .Path }}
Content: {{ .Content }}
Mode: {{ .Mode }}
--------------------------
{{- end }}
`
)

type prEntry struct {
	Path            string
	contentTemplate *template.Template
	Content         string
	Mode            string
}

// Remediator is the remediation engine for the Pull Request remediation type
type Remediator struct {
	ghCli      provifv1.GitHub
	gitCli     provifv1.Git
	actionType interfaces.ActionType

	titleTemplate *htmltemplate.Template
	bodyTemplate  *htmltemplate.Template
	entries       []prEntry
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

	ghCli, err := pbuild.GetGitHub(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get github client: %w", err)
	}

	gitCli, err := pbuild.GetGit()
	if err != nil {
		return nil, fmt.Errorf("failed to get git client: %w", err)
	}

	entries, err := prConfigToEntries(prCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR entries: %w", err)
	}

	return &Remediator{
		ghCli:      ghCli,
		gitCli:     gitCli,
		actionType: actionType,

		titleTemplate: titleTmpl,
		bodyTemplate:  bodyTmpl,
		entries:       entries,
	}, nil
}

func prConfigToEntries(prCfg *pb.RuleType_Definition_Remediate_PullRequestRemediation) ([]prEntry, error) {
	entries := make([]prEntry, len(prCfg.Contents))
	for i := range prCfg.Contents {
		cnt := prCfg.Contents[i]

		contentTemplate, err := util.ParseNewTextTemplate(&cnt.Content, fmt.Sprintf("Content[%d]", i))
		if err != nil {
			return nil, fmt.Errorf("cannot parse content template (index %d): %w", i, err)
		}

		mode := ghModeNonExecFile
		if cnt.Mode != nil {
			mode = *cnt.Mode
		}

		entries[i] = prEntry{
			Path:            cnt.Path,
			Mode:            mode,
			contentTemplate: contentTemplate,
		}
	}

	return entries, nil
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

	if err := r.expandContents(tmplParams); err != nil {
		return nil, fmt.Errorf("cannot expand contents: %w", err)
	}

	prFullBodyText, magicComment, err := r.getPrBodyText(tmplParams)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR full body text: %w", err)
	}

	var remErr error
	switch remAction {
	case interfaces.ActionOptOn:
		alreadyExists, err := prAlreadyExists(ctx, r.ghCli, repo, magicComment)
		if err != nil {
			return nil, fmt.Errorf("cannot check if PR already exists: %w", err)
		}
		if alreadyExists {
			zerolog.Ctx(ctx).Info().Msg("PR already exists, won't create a new one")
			return nil, nil
		}
		remErr = r.runGit(ctx, ingested.Fs, ingested.Storer, repo, title.String(), prFullBodyText)
	case interfaces.ActionOptDryRun:
		dryRun(title.String(), prFullBodyText, r.entries)
		remErr = nil
	case interfaces.ActionOptOff, interfaces.ActionOptUnknown:
		remErr = errors.New("unexpected action")
	}
	return nil, remErr
}

func dryRun(title, body string, entries []prEntry) {
	tmpl, err := template.New(dryRunTemplateName).Option("missingkey=error").Parse(dryRunTmpl)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	fmt.Printf("title:\n%s\n", title)
	fmt.Printf("body:\n%s\n", body)

	if err := tmpl.Execute(os.Stdout, entries); err != nil {
		log.Fatalf("Error executing template: %v", err)
	}
}

func (r *Remediator) runGit(
	ctx context.Context,
	fs billy.Filesystem,
	storer storage.Storer,
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

	logger.Debug().Msg("Getting authenticated user")
	u, err := r.ghCli.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("cannot get authenticated user: %w", err)
	}

	logger.Debug().Str("branch", branchBaseName(title)).Msg("Checking out branch")
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchBaseName(title)),
		Create: true,
	})
	if err != nil {
		return fmt.Errorf("cannot checkout branch: %w", err)
	}

	logger.Debug().Msg("Creating file entries")
	if err := r.createEntries(fs); err != nil {
		return fmt.Errorf("cannot create entries: %w", err)
	}

	logger.Debug().Msg("Staging changes")
	for _, entry := range r.entries {
		if _, err := wt.Add(entry.Path); err != nil {
			return fmt.Errorf("cannot add file %s: %w", entry.Path, err)
		}
	}

	logger.Debug().Msg("Committing changes")
	_, err = wt.Commit(title, &git.CommitOptions{
		Author: &object.Signature{
			Name:  u.GetName(),
			Email: u.GetEmail(),
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("cannot commit: %w", err)
	}

	refspec := refFromBranch(branchBaseName(title))

	var b bytes.Buffer
	err = repo.PushContext(ctx,
		&git.PushOptions{
			RemoteName: guessRemote(repo),
			Force:      true,
			RefSpecs: []config.RefSpec{
				config.RefSpec(
					fmt.Sprintf("+%s:%s", refspec, refspec),
				),
			},
			Auth: &githttp.BasicAuth{
				Username: u.GetName(),
				Password: r.ghCli.GetToken(),
			},
			Progress: &b,
		})
	if err != nil {
		return fmt.Errorf("cannot push: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msgf("Push output: %s", b.String())

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

func (r *Remediator) createEntries(fs billy.Filesystem) error {
	for _, entry := range r.entries {
		if err := writeEntry(entry, fs); err != nil {
			return fmt.Errorf("cannot write entry %s: %w", entry.Path, err)
		}
	}
	return nil
}

func writeEntry(entry prEntry, fs billy.Filesystem) error {
	if err := fs.MkdirAll(filepath.Dir(entry.Path), 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	f, err := fs.Create(entry.Path)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer f.Close()

	_, err = io.WriteString(f, entry.Content)
	if err != nil {
		return fmt.Errorf("cannot write to file: %w", err)
	}

	return nil
}

func refFromBranch(branchFrom string) string {
	return fmt.Sprintf("refs/heads/%s", branchFrom)
}

func (r *Remediator) expandContents(
	tmplParams *PrTemplateParams,
) error {
	for i := range r.entries {
		entry := &r.entries[i]
		content := new(bytes.Buffer)
		if err := entry.contentTemplate.Execute(content, tmplParams); err != nil {
			return fmt.Errorf("cannot execute content template (index %d): %w", i, err)
		}
		entry.Content = content.String()
	}

	return nil
}

func branchBaseName(prTitle string) string {
	baseName := dflBranchBaseName
	normalizedPrTitle := strings.ReplaceAll(strings.ToLower(prTitle), " ", "_")
	return fmt.Sprintf("%s_%s", baseName, normalizedPrTitle)
}

func (r *Remediator) contentSha1() (string, error) {
	var combinedContents string

	for i := range r.entries {
		if len(r.entries[i].Content) == 0 {
			// just making sure we call contentSha1() after expandContents()
			return "", fmt.Errorf("content (index %d) is empty", i)
		}
		combinedContents += r.entries[i].Path + r.entries[i].Content
	}

	// #nosec G401 - we're not using sha1 for crypto, only to quickly compare contents
	return fmt.Sprintf("%x", sha1.Sum([]byte(combinedContents))), nil
}

func (r *Remediator) prMagicComment() (string, error) {
	tmpl, err := template.New(prMagicTemplateName).Option("missingkey=error").Parse(prBodyMagicTemplate)
	if err != nil {
		return "", err
	}

	contentSha, err := r.contentSha1()
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

func (r *Remediator) getPrBodyText(tmplParams *PrTemplateParams) (string, string, error) {
	body := new(bytes.Buffer)
	if err := r.bodyTemplate.Execute(body, tmplParams); err != nil {
		return "", "", fmt.Errorf("cannot execute body template: %w", err)
	}

	magicComment, err := r.prMagicComment()
	if err != nil {
		return "", "", fmt.Errorf("cannot create PR magic comment: %w", err)
	}

	prFullBodyText, err := createReviewBody(body.String(), magicComment)
	if err != nil {
		return "", "", fmt.Errorf("cannot create PR full body text: %w", err)
	}

	return prFullBodyText, magicComment, nil
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
func prAlreadyExists(
	ctx context.Context,
	cli provifv1.GitHub,
	repo *pb.Repository,
	magicComment string,
) (bool, error) {
	// TODO(jakub): pagination
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
