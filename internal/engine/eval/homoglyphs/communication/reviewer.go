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

// Package communication contains the communication logic for the homoglyphs rule type
package communication

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// GhReviewPrHandler is a GitHub pull request review handler
type GhReviewPrHandler struct {
	logger zerolog.Logger

	ghClient provifv1.GitHub
	pr       *pb.PullRequest

	minderReview *github.PullRequestReview
	comments     []*github.DraftReviewComment
}

// NewGhReviewPrHandler creates a new GitHub pull request review handler
func NewGhReviewPrHandler(ghClient provifv1.GitHub) *GhReviewPrHandler {
	return &GhReviewPrHandler{
		ghClient: ghClient,
	}
}

// SubmitReview submits a review to a pull request
func (ra *GhReviewPrHandler) SubmitReview(ctx context.Context, reviewText string) error {
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

	if err := ra.submitReview(ctx, reviewText); err != nil {
		return fmt.Errorf("could not submit review: %w", err)
	}
	ra.logger.Debug().Msg("submitted review")
	return nil
}

// Hydrate hydrates the handler with a pull request
func (ra *GhReviewPrHandler) Hydrate(ctx context.Context, pr *pb.PullRequest) {
	logger := zerolog.Ctx(ctx).With().
		Int64("pull-number", pr.Number).
		Str("repo-owner", pr.RepoOwner).
		Str("repo-name", pr.RepoName).
		Logger()

	ra.logger = logger
	ra.pr = pr
	ra.comments = make([]*github.DraftReviewComment, 0)
	ra.minderReview = nil
}

// AddComment adds a comment to the review
func (ra *GhReviewPrHandler) AddComment(comment *github.DraftReviewComment) {
	ra.comments = append(ra.comments, comment)
}

// GetComments returns the comments of the review
func (ra *GhReviewPrHandler) GetComments() []*github.DraftReviewComment {
	return ra.comments
}

func (ra *GhReviewPrHandler) findPreviousReview(ctx context.Context) error {
	reviews, err := ra.ghClient.ListReviews(ctx, ra.pr.RepoOwner, ra.pr.RepoName, int(ra.pr.Number), nil)
	if err != nil {
		return fmt.Errorf("could not list reviews: %w", err)
	}

	ra.minderReview = nil
	for _, r := range reviews {
		if strings.HasPrefix(r.GetBody(), util.ReviewBodyMagicComment) && r.GetState() != "DISMISSED" {
			ra.minderReview = r
			break
		}
	}

	return nil
}

func (ra *GhReviewPrHandler) submitReview(ctx context.Context, reviewText string) error {
	body, err := util.CreateReviewBody(reviewText)
	if err != nil {
		return fmt.Errorf("could not create review body: %w", err)
	}

	review := &github.PullRequestReviewRequest{
		CommitID: github.String(ra.pr.CommitSha),
		Event:    github.String("COMMENT"),
		Comments: ra.comments,
		Body:     github.String(body),
	}

	_, err = ra.ghClient.CreateReview(
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

func (ra *GhReviewPrHandler) dismissReview(ctx context.Context) error {
	if ra.minderReview == nil {
		return nil
	}

	dismissReview := &github.PullRequestReviewDismissalRequest{
		Message: github.String(util.ReviewBodyDismissCommentText),
	}

	_, err := ra.ghClient.DismissReview(
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
