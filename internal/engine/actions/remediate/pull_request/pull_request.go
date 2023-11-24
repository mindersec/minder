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
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/google/go-github/v53/github"
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
	cli        provifv1.GitHub
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

	cli, err := pbuild.GetGitHub(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get github client: %w", err)
	}

	entries, err := prConfigToEntries(prCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR entries: %w", err)
	}

	return &Remediator{
		cli:        cli,
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
	_ *interfaces.Result,
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
		alreadyExists, err := prAlreadyExists(ctx, r.cli, repo, magicComment)
		if err != nil {
			return nil, fmt.Errorf("cannot check if PR already exists: %w", err)
		}
		if alreadyExists {
			zerolog.Ctx(ctx).Info().Msg("PR already exists, won't create a new one")
			return nil, nil
		}
		remErr = r.run(ctx, repo, title.String(), prFullBodyText, params.GetRule().Params.AsMap())
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

func (r *Remediator) run(
	ctx context.Context,
	repo *pb.Repository,
	title, body string,
	params map[string]any,
) error {
	branchFrom := getBranchFrom(ctx, params)

	commitShaFrom, err := r.getCommitShaFrom(ctx, repo, branchFrom)
	if err != nil {
		return fmt.Errorf("cannot get ref from: %w", err)
	}

	treeShaFrom, err := r.getTreeShaFrom(ctx, repo, commitShaFrom)
	if err != nil {
		return fmt.Errorf("cannot get tree from: %w", err)
	}

	tree, err := r.createTree(ctx, repo, treeShaFrom)
	if err != nil {
		return fmt.Errorf("error creating tree: %w", err)
	}

	newBranch, err := r.createOrUpdateBranch(ctx, repo, title, tree, commitShaFrom)
	if err != nil {
		return fmt.Errorf("error creating or updating branch: %w", err)
	}

	_, err = r.cli.CreatePullRequest(
		ctx, repo.GetOwner(), repo.GetName(),
		title, body,
		newBranch, dflBranchTo,
	)
	if err != nil {
		return fmt.Errorf("cannot create pull request: %w", err)
	}
	return nil
}

func (r *Remediator) getCommitShaFrom(ctx context.Context, repo *pb.Repository, branchFrom string) (string, error) {
	refName := refFromBranch(branchFrom)
	fromRef, err := r.cli.GetRef(ctx, repo.GetOwner(), repo.GetName(), refName)
	if err != nil {
		return "", fmt.Errorf("error getting commit ref: %w", err)
	}

	return fromRef.GetObject().GetSHA(), nil
}

func refFromBranch(branchFrom string) string {
	return fmt.Sprintf("refs/heads/%s", branchFrom)
}

func (r *Remediator) getTreeShaFrom(ctx context.Context, repo *pb.Repository, commitSha string) (string, error) {
	commit, err := r.cli.GetCommit(ctx, repo.GetOwner(), repo.GetName(), commitSha)
	if err != nil {
		return "", fmt.Errorf("error getting commit: %w", err)
	}
	return commit.GetTree().GetSHA(), nil
}

func (r *Remediator) createTree(ctx context.Context, repo *pb.Repository, treeShaFrom string) (*github.Tree, error) {
	treeEntries, err := r.createTreeEntries(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("error creating tree entries: %w", err)
	}

	tree, err := r.cli.CreateTree(ctx, repo.GetOwner(), repo.GetName(), treeShaFrom, treeEntries)
	if err != nil {
		return nil, fmt.Errorf("error creating tree: %w", err)
	}

	return tree, nil
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

func (r *Remediator) createTreeEntries(
	ctx context.Context,
	repo *pb.Repository,
) ([]*github.TreeEntry, error) {
	var treeEntries []*github.TreeEntry

	for i := range r.entries {
		entry := r.entries[i]
		blob := &github.Blob{
			Content:  github.String(entry.Content),
			Encoding: github.String("utf-8"),
		}
		newBlob, err := r.cli.CreateBlob(ctx, repo.GetOwner(), repo.GetName(), blob)
		if err != nil {
			return nil, fmt.Errorf("error creating blob: %w", err)
		}

		treeEntries = append(treeEntries, &github.TreeEntry{
			SHA:  newBlob.SHA,
			Path: github.String(entry.Path),
			Mode: github.String(entry.Mode),
			Type: github.String("blob"),
		})
	}

	return treeEntries, nil
}

func (r *Remediator) createOrUpdateBranch(
	ctx context.Context,
	repo *pb.Repository,
	title string,
	tree *github.Tree,
	commitShaFrom string,
) (string, error) {
	commit, err := r.cli.CreateCommit(ctx, repo.GetOwner(), repo.GetName(), title, tree, commitShaFrom)
	if err != nil {
		return "", fmt.Errorf("error creating commit: %w", err)
	}

	newBranch := refFromBranch(branchBaseName(title))
	_, err = r.cli.CreateRef(ctx, repo.GetOwner(), repo.GetName(), newBranch, *commit.SHA)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusUnprocessableEntity {
			_, ghErr := r.cli.UpdateRef(ctx, repo.GetOwner(), repo.GetName(), newBranch, *commit.SHA, true)
			if ghErr != nil {
				return "", fmt.Errorf("cannot update ref: %w", ghErr)
			}
		}
	}

	return newBranch, nil
}

func branchBaseName(prTitle string) string {
	baseName := dflBranchBaseName
	normalizedPrTitle := strings.ReplaceAll(strings.ToLower(prTitle), " ", "_")
	return fmt.Sprintf("%s_%s", baseName, normalizedPrTitle)
}

func getBranchFrom(ctx context.Context, params map[string]any) string {
	branchFrom, err := util.JQReadFrom[string](ctx, ".branch", params)
	if err != nil {
		zerolog.Ctx(ctx).Info().Msgf("error reading branchFrom from params, using default: %s", dflBranchFrom)
		branchFrom = dflBranchFrom
	}
	return branchFrom
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
