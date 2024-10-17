// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"context"
	"fmt"

	"github.com/google/go-github/v63/github"

	"github.com/mindersec/minder/internal/engine/eval/pr_actions"
	pbinternal "github.com/mindersec/minder/internal/proto"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

type prStatusHandler interface {
	trackVulnerableDep(
		ctx context.Context,
		dep *pbinternal.PrDependencies_ContextualDependency,
		vulnResp *VulnerabilityResponse,
		patch patchLocatorFormatter,
	) error
	submit(ctx context.Context) error
}

func newPrStatusHandler(
	ctx context.Context,
	action pr_actions.Action,
	pr *pb.PullRequest,
	client provifv1.GitHub,
) (prStatusHandler, error) {
	switch action {
	case pr_actions.ActionReviewPr:
		return newReviewPrHandler(ctx, pr, client)
	case pr_actions.ActionCommitStatus:
		return newCommitStatusPrHandler(ctx, pr, client)
	case pr_actions.ActionComment:
		return newReviewPrHandler(ctx, pr, client, withVulnsFoundReviewStatus(github.String("COMMENT")))
	case pr_actions.ActionProfileOnly:
		return newProfileOnlyPrHandler(), nil
	case pr_actions.ActionSummary:
		return newSummaryPrHandler(ctx, pr, client), nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}
