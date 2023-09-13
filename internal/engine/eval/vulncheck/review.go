// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
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
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/google/go-github/v53/github"
	"github.com/rs/zerolog"

	ghclient "github.com/stacklok/mediator/internal/providers/github"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const (
	reviewBodyMagicComment              = "<!-- mediator: pr-review-body -->"
	reviewBodyRequestChangesCommentText = `
Mediator found vulnerable dependencies in this PR. Either push an updated
version or accept the proposed changes. Note that accepting the changes will
include mediator as a co-author of this PR.
`
	reviewBodyCommentText = `
Mediator analyzed this PR and found no vulnerable dependencies.
`
	reviewBodyDismissCommentText = `
Previous mediator review was dismissed because the PR was updated.
`
)

const (
	reviewTemplateName = "reviewBody"
	reviewTmplStr      = "{{.MagicComment}}\n\n{{.ReviewText}}"
)

type reviewTemplateData struct {
	MagicComment string
	ReviewText   string
}

func createReviewBody(magicComment, reviewText string) (string, error) {
	// Create and parse the template
	tmpl, err := template.New(reviewTemplateName).Parse(reviewTmplStr)
	if err != nil {
		return "", err
	}

	// Define the data for the template
	data := reviewTemplateData{
		MagicComment: magicComment,
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
	lineToChange      int
	leadingWhitespace int
}

func countLeadingWhitespace(line string) int {
	count := 0
	for _, ch := range line {
		if ch != ' ' && ch != '\t' {
			return count
		}
		count++
	}
	return count
}

func locateDepInPr(
	_ context.Context,
	client ghclient.RestAPI,
	dep *pb.PrDependencies_ContextualDependency,
) (*reviewLocation, error) {
	req, err := client.NewRequest("GET", dep.File.PatchUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	// TODO:(jakub) I couldn't make this work with the GH client
	netClient := &http.Client{}
	resp, _ := netClient.Do(req)
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
		pkgName := fmt.Sprintf(`"%s": {`, dep.Dep.Name)
		if strings.Contains(line, pkgName) {
			loc.leadingWhitespace = countLeadingWhitespace(line)
			loc.lineToChange = i + 1
		}
	}

	return &loc, nil
}

type reviewPrHandler struct {
	cli ghclient.RestAPI
	pr  *pb.PullRequest

	mediatorReview *github.PullRequestReview

	comments []*github.DraftReviewComment
	status   *string
	text     *string

	logger zerolog.Logger
}

func newReviewPrHandler(
	ctx context.Context,
	pr *pb.PullRequest,
	cli ghclient.RestAPI,
) (prStatusHandler, error) {
	if pr == nil {
		return nil, fmt.Errorf("pr was nil, can't review")
	}

	logger := zerolog.Ctx(ctx).With().
		Int32("pull-number", pr.Number).
		Str("repo-owner", pr.RepoOwner).
		Str("repo-name", pr.RepoName).
		Logger()

	return &reviewPrHandler{
		cli:      cli,
		pr:       pr,
		comments: []*github.DraftReviewComment{},
		logger:   logger,
	}, nil
}

func (ra *reviewPrHandler) trackVulnerableDep(
	ctx context.Context,
	dep *pb.PrDependencies_ContextualDependency,
	patch patchFormatter,
) error {
	location, err := locateDepInPr(ctx, ra.cli, dep)
	if err != nil {
		return fmt.Errorf("could not locate dependency in PR: %w", err)
	}

	comment := patch.IndentedString(location.leadingWhitespace)
	body := fmt.Sprintf("```suggestion\n"+"%s\n"+"```\n", comment)

	reviewComment := &github.DraftReviewComment{
		Path:      github.String(dep.File.Name),
		Position:  nil,
		StartLine: github.Int(location.lineToChange),
		Line:      github.Int(location.lineToChange + 3), // TODO(jakub): Need to count the lines from the patch
		Body:      github.String(body),
	}
	ra.comments = append(ra.comments, reviewComment)

	return nil
}

func (ra *reviewPrHandler) submit(ctx context.Context) error {
	if err := ra.findPreviousReview(ctx); err != nil {
		return fmt.Errorf("could not find previous review: %w", err)
	}

	if ra.mediatorReview != nil {
		err := ra.dismissReview(ctx)
		if err != nil {
			ra.logger.Error().Err(err).
				Int64("review-id", ra.mediatorReview.GetID()).
				Msg("could not dismiss previous review")
		}
		ra.logger.Debug().
			Int64("review-id", ra.mediatorReview.GetID()).
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
		ra.status = github.String("REQUEST_CHANGES")
		ra.text = github.String(reviewBodyRequestChangesCommentText)
	} else {
		// if this pass produced no comments, resolve the mediator review
		ra.status = github.String("COMMENT")
		ra.text = github.String(reviewBodyCommentText)
	}
}

func (ra *reviewPrHandler) findPreviousReview(ctx context.Context) error {
	reviews, err := ra.cli.ListReviews(ctx, ra.pr.RepoOwner, ra.pr.RepoName, int(ra.pr.Number), nil)
	if err != nil {
		return fmt.Errorf("could not list reviews: %w", err)
	}

	ra.mediatorReview = nil
	for _, r := range reviews {
		if strings.HasPrefix(r.GetBody(), reviewBodyMagicComment) && r.GetState() != "DISMISSED" {
			ra.mediatorReview = r
			break
		}
	}

	return nil
}

func (ra *reviewPrHandler) submitReview(ctx context.Context) error {
	body, err := createReviewBody(reviewBodyMagicComment, *ra.text)
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
	if ra.mediatorReview == nil {
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
		ra.mediatorReview.GetID(),
		dismissReview)
	if err != nil {
		return fmt.Errorf("could not dismiss review: %w", err)
	}
	return nil
}
