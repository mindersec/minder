// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request_comment

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// alertFlusher aggregates a list of comments and flushes them to the PR
// as a single comment. The idea is that we can aggregate multiple alerts
// into a single comment without needing to flood the PR with multiple comments.
// This is only instantiated once; the first creation is the only one that will
// be used.
type alertFlusher struct {
	props     *properties.Properties
	commitSha string
	commenter provifv1.PullRequestCommenter
}

func newAlertFlusher(props *properties.Properties, commitSha string, commenter provifv1.PullRequestCommenter) *alertFlusher {
	return &alertFlusher{
		props:     props,
		commitSha: commitSha,
		commenter: commenter,
	}
}

func (a *alertFlusher) Flush(ctx context.Context, items ...any) error {
	logger := zerolog.Ctx(ctx)

	aggregatedCommentBody := paragraph(title1("Minder Alerts"))

	// iterate and aggregate
	for _, item := range items {
		fp, ok := item.(*provifv1.PullRequestCommentInfo)
		if !ok {
			logger.Error().Msgf("expected PullRequestCommentInfo, got %T", item)
			continue
		}

		aggregatedCommentBody += paragraph(alert(fp.Header, fp.Body))
	}

	_, err := a.commenter.CommentOnPullRequest(ctx, a.props, provifv1.PullRequestCommentInfo{
		Commit: a.commitSha,
		Body:   aggregatedCommentBody,
	})
	if err != nil {
		return fmt.Errorf("error creating PR review: %w", err)
	}

	return nil
}
