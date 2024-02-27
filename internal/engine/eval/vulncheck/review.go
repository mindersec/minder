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

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"io"
	"strings"
	"text/template"

	"github.com/google/go-github/v56/github"
	"github.com/rs/zerolog"

	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	reviewBodyMagicComment = "<!-- minder: pr-review-body -->"
	commitStatusContext    = "minder.stacklok.dev/pr-vulncheck"
	vulnsFoundText         = `
Minder found vulnerable dependencies in this PR. Either push an updated
version or accept the proposed changes. Note that accepting the changes will
include Minder as a co-author of this PR.
`
	vulnsFoundTextShort = `
Vulnerable dependencies found.
`
	noVulsFoundText = `
Minder analyzed this PR and found no vulnerable dependencies.
`
	reviewBodyDismissCommentText = `
Previous Minder review was dismissed because the PR was updated.
`
	vulnFoundWithNoPatch = "Vulnerability found, but no patched version exists yet."
)

const (
	reviewTemplateName = "reviewBody"
	reviewTmplStr      = "{{.MagicComment}}\n\n{{.ReviewText}}"
)

const (
	tabSize = 8
)

type reviewTemplateData struct {
	MagicComment string
	ReviewText   string
}

func createReviewBody(reviewText string) (string, error) {
	// Create and parse the template
	tmpl, err := template.New(reviewTemplateName).Option("missingkey=error").Parse(reviewTmplStr)
	if err != nil {
		return "", err
	}

	// Define the data for the template
	data := reviewTemplateData{
		MagicComment: reviewBodyMagicComment,
		ReviewText:   reviewText,
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type reviewLocation struct {
	line              string
	lineToChange      int
	leadingWhitespace int
}

func countLeadingWhitespace(line string) int {
	count := 0
	for _, ch := range line {
		if ch == '\t' {
			count += tabSize
			continue
		}
		if ch != ' ' {
			return count
		}
		count++
	}
	return count
}

func locateDepInPr(
	ctx context.Context,
	client provifv1.GitHub,
	dep *pb.PrDependencies_ContextualDependency,
	patch patchLocatorFormatter,
) (*reviewLocation, error) {
	req, err := client.NewRequest("GET", dep.File.PatchUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	resp, err := client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	loc := reviewLocation{}
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if patch.LineHasDependency(line) {
			loc.leadingWhitespace = countLeadingWhitespace(line)
			loc.lineToChange = i + 1
			loc.line = line
			break
		}
	}

	if loc.lineToChange == 0 {
		return nil, fmt.Errorf("could not locate dependency in PR")
	}

	return &loc, nil
}

func reviewBodyWithSuggestion(comment string) string {
	return fmt.Sprintf("```suggestion\n%s\n```\n", comment)
}

type reviewPrHandler struct {
	cli provifv1.GitHub
	pr  *pb.PullRequest

	minderReview *github.PullRequestReview
	failStatus   *string

	comments []*github.DraftReviewComment
	status   *string
	text     *string

	logger zerolog.Logger
}

type reviewPrHandlerOption func(*reviewPrHandler)

// WithSetReviewStatus is an option to set the vulnsFoundReviewStatus field of reviewPrHandler.
func withVulnsFoundReviewStatus(status *string) reviewPrHandlerOption {
	return func(r *reviewPrHandler) {
		r.failStatus = status
	}
}

func newReviewPrHandler(
	ctx context.Context,
	pr *pb.PullRequest,
	cli provifv1.GitHub,
	opts ...reviewPrHandlerOption,
) (*reviewPrHandler, error) {
	if pr == nil {
		return nil, fmt.Errorf("pr was nil, can't review")
	}

	logger := zerolog.Ctx(ctx).With().
		Int64("pull-number", pr.Number).
		Str("repo-owner", pr.RepoOwner).
		Str("repo-name", pr.RepoName).
		Logger()

	cliUser, err := cli.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get authenticated user: %w", err)
	}

	// if the user wants minder to request changes on a pull request, they need to
	// be different identities
	var failStatus *string
	if pr.AuthorId == cliUser.GetID() {
		failStatus = github.String("COMMENT")
		logger.Debug().Msg("author is the same as the authenticated user, can only comment")
	} else {
		failStatus = github.String("REQUEST_CHANGES")
		logger.Debug().Msg("author is different than the authenticated user, can request changes")
	}

	handler := &reviewPrHandler{
		cli:        cli,
		pr:         pr,
		comments:   []*github.DraftReviewComment{},
		logger:     logger,
		failStatus: failStatus,
	}

	for _, opt := range opts {
		opt(handler)
	}

	return handler, nil
}

func (ra *reviewPrHandler) trackVulnerableDep(
	ctx context.Context,
	dep *pb.PrDependencies_ContextualDependency,
	_ *VulnerabilityResponse,
	patch patchLocatorFormatter,
) error {
	location, err := locateDepInPr(ctx, ra.cli, dep, patch)
	if err != nil {
		return fmt.Errorf("could not locate dependency in PR: %w", err)
	}

	var body string
	var lineTo int
	if patch.HasPatchedVersion() {
		comment := patch.IndentedString(location.leadingWhitespace, location.line, dep.Dep)
		body = reviewBodyWithSuggestion(comment)
		lineTo = len(strings.Split(comment, "\n")) - 1
	} else {
		body = vulnFoundWithNoPatch
	}

	reviewComment := &github.DraftReviewComment{
		Path: github.String(dep.File.Name),
		Body: github.String(body),
	}

	if lineTo > 0 {
		reviewComment.StartLine = github.Int(location.lineToChange)
		reviewComment.Line = github.Int(location.lineToChange + lineTo)
	} else {
		reviewComment.Line = github.Int(location.lineToChange)
	}

	ra.comments = append(ra.comments, reviewComment)

	ra.logger.Debug().
		Str("dep-name", dep.Dep.Name).
		Msg("vulnerable dependency found")

	return nil
}

func (ra *reviewPrHandler) submit(ctx context.Context) error {
	if err := ra.findPreviousReview(ctx); err != nil {
		return fmt.Errorf("could not find previous review: %w", err)
	}

	if ra.minderReview != nil {
		if ra.minderReview.CommitID != nil && *ra.minderReview.CommitID == ra.pr.CommitSha {
			// if the previous review was on the same commit, keep it
			ra.logger.Debug().
				Int64("review-id", ra.minderReview.GetID()).
				Msg("previous review was on the same commit, will keep it")
			return nil
		}

		err := ra.dismissReview(ctx)
		if err != nil {
			ra.logger.Error().Err(err).
				Int64("review-id", ra.minderReview.GetID()).
				Msg("could not dismiss previous review")
		}
		ra.logger.Debug().
			Int64("review-id", ra.minderReview.GetID()).
			Msg("dismissed previous review")
	}

	// either there are changes to request or just send the first review mentioning that everything is ok
	ra.setStatus()
	if err := ra.submitReview(ctx); err != nil {
		return fmt.Errorf("could not submit review: %w", err)
	}
	ra.logger.Debug().Msg("submitted review")
	return nil
}

func (ra *reviewPrHandler) setStatus() {
	if len(ra.comments) > 0 {
		// if this pass produced comments, request changes
		ra.text = github.String(vulnsFoundText)
		ra.status = ra.failStatus
		ra.logger.Debug().Msg("vulnerabilities found")
	} else {
		// if this pass produced no comments, resolve the minder review
		ra.status = github.String("COMMENT")
		ra.text = github.String(noVulsFoundText)
		ra.logger.Debug().Msg("no vulnerabilities found")
	}

	ra.logger.Debug().Str("status", *ra.status).Msg("will set review status")
}

func (ra *reviewPrHandler) findPreviousReview(ctx context.Context) error {
	reviews, err := ra.cli.ListReviews(ctx, ra.pr.RepoOwner, ra.pr.RepoName, int(ra.pr.Number), nil)
	if err != nil {
		return fmt.Errorf("could not list reviews: %w", err)
	}

	ra.minderReview = nil
	for _, r := range reviews {
		if strings.HasPrefix(r.GetBody(), reviewBodyMagicComment) && r.GetState() != "DISMISSED" {
			ra.minderReview = r
			break
		}
	}

	return nil
}

func (ra *reviewPrHandler) submitReview(ctx context.Context) error {
	body, err := createReviewBody(*ra.text)
	if err != nil {
		return fmt.Errorf("could not create review body: %w", err)
	}

	review := &github.PullRequestReviewRequest{
		CommitID: github.String(ra.pr.CommitSha),
		Event:    ra.status,
		Comments: ra.comments,
		Body:     github.String(body),
	}

	_, err = ra.cli.CreateReview(
		ctx,
		ra.pr.RepoOwner,
		ra.pr.RepoName,
		int(ra.pr.Number),
		review,
	)
	if err != nil {
		return fmt.Errorf("could not create review: %w", err)
	}

	return nil
}

func (ra *reviewPrHandler) dismissReview(ctx context.Context) error {
	if ra.minderReview == nil {
		return nil
	}

	dismissReview := &github.PullRequestReviewDismissalRequest{
		Message: github.String(reviewBodyDismissCommentText),
	}

	_, err := ra.cli.DismissReview(
		ctx,
		ra.pr.RepoOwner,
		ra.pr.RepoName,
		int(ra.pr.Number),
		ra.minderReview.GetID(),
		dismissReview)
	if err != nil {
		return fmt.Errorf("could not dismiss review: %w", err)
	}
	return nil
}

type commitStatusPrHandler struct {
	// embed the reviewPrHandler to automatically satisfy the prStatusHandler interface
	reviewPrHandler
}

func newCommitStatusPrHandler(
	ctx context.Context,
	pr *pb.PullRequest,
	client provifv1.GitHub,
) (prStatusHandler, error) {
	// create a reviewPrHandler and embed it in the commitStatusPrHandler
	rph, err := newReviewPrHandler(
		ctx,
		pr,
		client,
		withVulnsFoundReviewStatus(github.String("COMMENT")),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create review handler: %w", err)
	}

	return &commitStatusPrHandler{
		reviewPrHandler: *rph,
	}, nil
}

func (csh *commitStatusPrHandler) submit(ctx context.Context) error {
	// first submit the review, we force the status to be COMMENT to not block
	if err := csh.reviewPrHandler.submit(ctx); err != nil {
		return fmt.Errorf("could not submit review: %w", err)
	}

	// next either pass or fail the commit status to eventually block the PR
	if err := csh.setCommitStatus(ctx); err != nil {
		return fmt.Errorf("could not set commit status: %w", err)
	}

	return nil
}

func (csh *commitStatusPrHandler) setCommitStatus(
	ctx context.Context,
) error {
	commitStatus := &github.RepoStatus{
		Context: github.String(commitStatusContext),
	}

	if len(csh.comments) > 0 {
		commitStatus.State = github.String("failure")
		commitStatus.Description = github.String(vulnsFoundTextShort)
	} else {
		commitStatus.State = github.String("success")
		commitStatus.Description = github.String(noVulsFoundText)
	}

	csh.logger.Debug().
		Str("commit-status", commitStatus.String()).
		Str("commit-sha", csh.pr.CommitSha).
		Msg("setting commit status")

	_, err := csh.cli.SetCommitStatus(ctx, csh.pr.RepoOwner, csh.pr.RepoName, csh.pr.CommitSha, commitStatus)
	return err
}

// summaryPrHandler is a prStatusHandler that adds a summary text to the PR as a comment.
type summaryPrHandler struct {
	cli provifv1.GitHub
	pr  *pb.PullRequest

	logger      zerolog.Logger
	trackedDeps []dependencyVulnerabilities
	headerTmpl  *htmltemplate.Template
	rowsTmpl    *htmltemplate.Template
}

const (
	tableVulnerabilitiesHeaderName = "vulnerabilitiesTableHeader"
	tableVulnerabilitiesHeader     = `### Summary of vulnerabilities found
Minder found the following vulnerabilities in this PR:
<table>
  <tr>
    <th>Ecosystem</th>
    <th>Name</th>
    <th>Version</th>
    <th>Vulnerability ID</th>
    <th>Summary</th>
    <th>Introduced</th>
    <th>Fixed</th>
  </tr>
`
	tableVulnerabilitiesRowsName = "vulnerabilitiesTableRow"
	tableVulnerabilitiesRows     = `
  {{ range .Vulnerabilities }}
  <tr>
    <td>{{ $.Ecosystem }}</td>
    <td>{{ $.Name }}</td>
    <td>{{ $.Version }}</td>
    <td>{{ .ID }}</td>
    <td>{{ .Summary }}</td>
    <td>{{ .Introduced }}</td>
    <td>{{ .Fixed }}</td>
  </tr>
  {{ end }}
`
	tableVulnerabilitiesFooter = "</table>"
)

type dependencyVulnerabilities struct {
	Dependency      *pb.Dependency
	Vulnerabilities []Vulnerability
}

func (sph *summaryPrHandler) trackVulnerableDep(
	_ context.Context,
	dep *pb.PrDependencies_ContextualDependency,
	vulnResp *VulnerabilityResponse,
	_ patchLocatorFormatter,
) error {
	sph.trackedDeps = append(sph.trackedDeps, dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
	})
	return nil
}

func (sph *summaryPrHandler) submit(ctx context.Context) error {
	summary, err := sph.generateSummary()
	if err != nil {
		return fmt.Errorf("could not generate summary: %w", err)
	}

	err = sph.cli.CreateComment(ctx, sph.pr.GetRepoOwner(), sph.pr.GetRepoName(), int(sph.pr.GetNumber()), summary)
	if err != nil {
		return fmt.Errorf("could not create comment: %w", err)
	}

	return nil
}

func (sph *summaryPrHandler) generateSummary() (string, error) {
	var summary strings.Builder
	if len(sph.trackedDeps) == 0 {
		summary.WriteString(noVulsFoundText)
		return summary.String(), nil
	}

	var headerBuf bytes.Buffer
	if err := sph.headerTmpl.Execute(&headerBuf, nil); err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}
	summary.WriteString(headerBuf.String())

	for i := range sph.trackedDeps {
		var rowBuf bytes.Buffer

		if err := sph.rowsTmpl.Execute(&rowBuf, struct {
			Ecosystem       string
			Name            string
			Version         string
			Vulnerabilities []Vulnerability
		}{
			Ecosystem:       sph.trackedDeps[i].Dependency.Ecosystem.AsString(),
			Name:            sph.trackedDeps[i].Dependency.Name,
			Version:         sph.trackedDeps[i].Dependency.Version,
			Vulnerabilities: sph.trackedDeps[i].Vulnerabilities,
		}); err != nil {
			return "", fmt.Errorf("could not execute template: %w", err)
		}
		summary.WriteString(rowBuf.String())
	}
	summary.WriteString(tableVulnerabilitiesFooter)

	return summary.String(), nil
}

func newSummaryPrHandler(
	ctx context.Context,
	pr *pb.PullRequest,
	cli provifv1.GitHub,
) (prStatusHandler, error) {
	logger := zerolog.Ctx(ctx).With().
		Int64("pull-number", pr.Number).
		Str("repo-owner", pr.RepoOwner).
		Str("repo-name", pr.RepoName).
		Logger()

	headerTmpl, err := htmltemplate.New(tableVulnerabilitiesHeaderName).Parse(tableVulnerabilitiesHeader)
	if err != nil {
		return nil, fmt.Errorf("could not parse dependency template: %w", err)
	}
	rowsTmpl, err := htmltemplate.New(tableVulnerabilitiesRowsName).Parse(tableVulnerabilitiesRows)
	if err != nil {
		return nil, fmt.Errorf("could not parse vulnerability template: %w", err)
	}

	return &summaryPrHandler{
		cli:         cli,
		pr:          pr,
		logger:      logger,
		headerTmpl:  headerTmpl,
		rowsTmpl:    rowsTmpl,
		trackedDeps: make([]dependencyVulnerabilities, 0),
	}, nil
}

// just satisfies the interface but really does nothing. Useful for testing.
type profileOnlyPrHandler struct{}

func (profileOnlyPrHandler) trackVulnerableDep(
	_ context.Context,
	_ *pb.PrDependencies_ContextualDependency,
	_ *VulnerabilityResponse,
	_ patchLocatorFormatter,
) error {
	return nil
}

func (profileOnlyPrHandler) submit(_ context.Context) error {
	return nil
}

func newProfileOnlyPrHandler() prStatusHandler {
	return &profileOnlyPrHandler{}
}
