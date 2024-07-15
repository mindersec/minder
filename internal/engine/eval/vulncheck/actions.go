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

	"github.com/google/go-github/v61/github"

	"github.com/stacklok/minder/internal/engine/eval/pr_actions"
	pbinternal "github.com/stacklok/minder/internal/proto"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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
