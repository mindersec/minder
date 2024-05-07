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
// Package rule provides the CLI subcommand for managing rules

// Package remediate provides necessary interfaces and implementations for
// remediating rules.
package remediate

import (
	"fmt"

	"github.com/stacklok/minder/internal/engine/actions/remediate/gh_branch_protect"
	"github.com/stacklok/minder/internal/engine/actions/remediate/noop"
	"github.com/stacklok/minder/internal/engine/actions/remediate/pull_request"
	"github.com/stacklok/minder/internal/engine/actions/remediate/rest"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ActionType is the type of the remediation engine
const ActionType engif.ActionType = "remediate"

// NewRuleRemediator creates a new rule remediator
func NewRuleRemediator(rt *pb.RuleType, pbuild *providers.ProviderBuilder) (engif.Action, error) {
	rem := rt.Def.GetRemediate()
	if rem == nil {
		return noop.NewNoopRemediate(ActionType)
	}

	// nolint:revive // let's keep the switch here, it would be nicer to extend a switch in the future
	switch rem.GetType() {
	case rest.RemediateType:
		client, err := pbuild.GetHTTP()
		if err != nil {
			return nil, fmt.Errorf("could not instantiate provider: %w", err)
		}
		if rem.GetRest() == nil {
			return nil, fmt.Errorf("remediations engine missing rest configuration")
		}
		return rest.NewRestRemediate(ActionType, rem.GetRest(), client)

	case gh_branch_protect.RemediateType:
		client, err := pbuild.GetGitHub()
		if err != nil {
			return nil, fmt.Errorf("could not instantiate provider: %w", err)
		}
		if rem.GetGhBranchProtection() == nil {
			return nil, fmt.Errorf("remediations engine missing gh_branch_protection configuration")
		}
		return gh_branch_protect.NewGhBranchProtectRemediator(ActionType, rem.GetGhBranchProtection(), client)

	case pull_request.RemediateType:
		client, err := pbuild.GetGitHub()
		if err != nil {
			return nil, fmt.Errorf("could not instantiate provider: %w", err)
		}
		if rem.GetPullRequest() == nil {
			return nil, fmt.Errorf("remediations engine missing pull request configuration")
		}

		return pull_request.NewPullRequestRemediate(ActionType, rem.GetPullRequest(), client)
	}

	return nil, fmt.Errorf("unknown remediation type: %s", rem.GetType())
}
