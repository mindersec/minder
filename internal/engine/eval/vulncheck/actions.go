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

	"github.com/google/go-github/v53/github"

	ghclient "github.com/stacklok/mediator/internal/providers/github"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

type prStatusHandler interface {
	trackVulnerableDep(
		ctx context.Context,
		dep *pb.PrDependencies_ContextualDependency,
		patch patchFormatter,
	) error
	submit(ctx context.Context) error
}

func newPrStatusHandler(
	ctx context.Context,
	action action,
	pr *pb.PullRequest,
	client ghclient.RestAPI,
) (prStatusHandler, error) {
	switch action {
	case actionReviewPr:
		return newReviewPrHandler(ctx, pr, client)
	case actionCommitStatus:
		return newCommitStatusPrHandler(ctx, pr, client)
	case actionComment:
		return newReviewPrHandler(ctx, pr, client, withVulnsFoundReviewStatus(github.String("COMMENT")))
	case actionPolicyOnly:
		return newPolicyOnlyPrHandler(), nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}
