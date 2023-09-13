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
	"context"
	"fmt"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
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
	case actionRejectPr:
		return newReviewPrHandler(ctx, pr, client)
	case actionComment, actionPolicyOnly:
		return nil, fmt.Errorf("action %s not implemented", action)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}
