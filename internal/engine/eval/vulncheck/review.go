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
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/go-github/v56/github"
	"github.com/rs/zerolog"

	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	commitStatusContext = "minder.stacklok.dev/pr-vulncheck"
)

const (
	tabSize = 8
)

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

	trackedDeps []dependencyVulnerabilities

	authorizedUser     int64
	minderStatusReport *github.IssueComment
	comments           []*github.DraftReviewComment

	status     *string
	text       *string
	failStatus *string

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
	cliUserId, err := cli.GetUserId(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get authenticated user: %w", err)
	}

	// if the user wants minder to request changes on a pull request, they need to
	// be different identities
	var failStatus *string
	if pr.AuthorId == cliUserId {
		failStatus = github.String("COMMENT")
		logger.Debug().Msg("author is the same as the authenticated user, can only comment")
	} else {
		failStatus = github.String("REQUEST_CHANGES")
		logger.Debug().Msg("author is different than the authenticated user, can request changes")
	}

	handler := &reviewPrHandler{
		cli:            cli,
		pr:             pr,
		comments:       []*github.DraftReviewComment{},
		logger:         logger,
		failStatus:     failStatus,
		trackedDeps:    []dependencyVulnerabilities{},
		authorizedUser: cliUserId,
	}

	for _, opt := range opts {
		opt(handler)
	}

	return handler, nil
}

func (ra *reviewPrHandler) trackVulnerableDep(
	ctx context.Context,
	dep *pb.PrDependencies_ContextualDependency,
	vulnResp *VulnerabilityResponse,
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

	ra.trackedDeps = append(ra.trackedDeps, dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    patch.GetPatchedVersion(),
	})

	return nil
}

func (ra *reviewPrHandler) submit(ctx context.Context) error {
	if err := ra.findPreviousStatusComment(ctx); err != nil {
		return fmt.Errorf("could not find previous status comment: %w", err)
	}

	ra.setStatus()

	mci := ra.getMagicCommentInfo()

	var err error
	mci.ReviewID, err = ra.submitReview(ctx, mci)
	if err != nil {
		// this should be fatal. In case we can't submit the review, we can't proceed
		return fmt.Errorf("could not submit review: %w", err)
	}
	ra.logger.Debug().Msg("submitted review")

	// If no status comment exists, post one immediately
	if err := ra.updateStatusReportComment(ctx, mci); err != nil {
		return fmt.Errorf("could not create status report: %w", err)
	}

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

func (ra *reviewPrHandler) findPreviousStatusComment(ctx context.Context) error {
	comments, err := ra.cli.ListIssueComments(ctx, ra.pr.RepoOwner, ra.pr.RepoName, int(ra.pr.Number),
		&github.IssueListCommentsOptions{
			Sort:      github.String("updated"),
			Direction: github.String("asc"),
		})
	if err != nil {
		return fmt.Errorf("could not list comments: %w", err)
	}

	ra.minderStatusReport = nil
	for _, comment := range comments {
		isMinder := comment.GetUser().GetID() == ra.authorizedUser
		if isMinder && strings.HasPrefix(comment.GetBody(), statusBodyMagicCommentPrefix) {
			ra.minderStatusReport = comment
			break
		}
	}

	return nil
}

func (ra *reviewPrHandler) getMagicCommentInfo() magicCommentInfo {
	if ra.minderStatusReport == nil {
		return magicCommentInfo{}
	}

	mci, err := extractContentShaAndReviewID(ra.minderStatusReport.GetBody())
	if err != nil {
		ra.logger.Warn().Msg("could not extract content sha and review id from previous status comment")
		// non-fatal error, we can still post the status report
		// and worst case we will add a duplicate comment
	}
	return mci
}

func (ra *reviewPrHandler) submitReview(ctx context.Context, mci magicCommentInfo) (int64, error) {
	// if the previous review was on the same commit, keep it
	if mci.ContentSha == ra.pr.GetCommitSha() {
		ra.logger.Debug().
			Int64("review-id", mci.ReviewID).
			Msg("previous review was on the same commit, will keep it")
		return 0, nil
	}

	if err := ra.dismissReview(ctx, mci); err != nil {
		ra.logger.Warn().Msg("could not dismiss previous review")
	}

	return ra.createReview(ctx)
}

func (ra *reviewPrHandler) createReview(ctx context.Context) (int64, error) {
	var err error

	if len(ra.comments) == 0 {
		return 0, nil
	}

	review := &github.PullRequestReviewRequest{
		CommitID: github.String(ra.pr.CommitSha),
		Event:    ra.status,
		Comments: ra.comments,
	}

	r, err := ra.cli.CreateReview(
		ctx,
		ra.pr.RepoOwner,
		ra.pr.RepoName,
		int(ra.pr.Number),
		review,
	)
	if err != nil {
		return 0, fmt.Errorf("could not create review: %w", err)
	}

	return r.GetID(), nil
}

func (ra *reviewPrHandler) dismissReview(ctx context.Context, mci magicCommentInfo) error {
	if mci.ReviewID == 0 {
		ra.logger.Debug().Msg("no previous review to dismiss")
		return nil
	}

	if ra.pr.GetAuthorId() == ra.authorizedUser {
		ra.logger.Warn().Msg("author is the same as the authenticated user, can't dismiss")
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
		mci.ReviewID,
		dismissReview)
	if err != nil {
		return fmt.Errorf("could not dismiss review: %w", err)
	}
	return nil
}

func (ra *reviewPrHandler) updateStatusReportComment(ctx context.Context, mci magicCommentInfo) error {
	report := &statusReport{
		StatusText:          *ra.text,
		CommitSHA:           ra.pr.CommitSha,
		TrackedDependencies: ra.trackedDeps,
		ReviewID:            mci.ReviewID,
	}

	statusBody, err := report.render()
	if err != nil {
		return fmt.Errorf("failed to render status from templates: %w", err)
	}

	if ra.minderStatusReport == nil {
		if ra.minderStatusReport, err = ra.cli.CreateIssueComment(
			ctx,
			ra.pr.RepoOwner,
			ra.pr.RepoName,
			int(ra.pr.GetNumber()),
			statusBody,
		); err != nil {
			return fmt.Errorf("failed to create minder status report comment: %w", err)
		}

		return nil
	}

	if mci.ContentSha != ra.pr.GetCommitSha() {
		if err := ra.cli.UpdateIssueComment(
			ctx,
			ra.pr.RepoOwner,
			ra.pr.RepoName,
			ra.minderStatusReport.GetID(),
			statusBody,
		); err != nil {
			return fmt.Errorf("failed to update minder status report comment: %w", err)
		}
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
		csh.logger.Error().Err(err).Msg("could not submit review")
		// since in this case the mechanism that blocks the PR is the commit status,
		// we should not return an error here but try to set the commit status anyway
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
}

type dependencyVulnerabilities struct {
	Dependency      *pb.Dependency
	Vulnerabilities []Vulnerability
	PatchVersion    string
}

func (sph *summaryPrHandler) trackVulnerableDep(
	_ context.Context,
	dep *pb.PrDependencies_ContextualDependency,
	vulnResp *VulnerabilityResponse,
	patch patchLocatorFormatter,
) error {
	sph.trackedDeps = append(sph.trackedDeps, dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    patch.GetPatchedVersion(),
	})
	return nil
}

func (sph *summaryPrHandler) submit(ctx context.Context) error {
	report := &vulnSummaryReport{
		TrackedDependencies: sph.trackedDeps,
	}

	summary, err := report.render()
	if err != nil {
		return fmt.Errorf("could not generate summary: %w", err)
	}

	_, err = sph.cli.CreateIssueComment(ctx, sph.pr.GetRepoOwner(), sph.pr.GetRepoName(), int(sph.pr.GetNumber()), summary)
	if err != nil {
		return fmt.Errorf("could not create comment: %w", err)
	}

	return nil
}

func newSummaryPrHandler(
	ctx context.Context,
	pr *pb.PullRequest,
	cli provifv1.GitHub,
) *summaryPrHandler {
	logger := zerolog.Ctx(ctx).With().
		Int64("pull-number", pr.Number).
		Str("repo-owner", pr.RepoOwner).
		Str("repo-name", pr.RepoName).
		Logger()

	return &summaryPrHandler{
		cli:         cli,
		pr:          pr,
		logger:      logger,
		trackedDeps: make([]dependencyVulnerabilities, 0),
	}
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
