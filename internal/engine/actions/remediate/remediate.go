// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package remediate provides necessary interfaces and implementations for
// remediating rules.
package remediate

import (
	"errors"
	"fmt"

	"github.com/mindersec/minder/internal/engine/actions/remediate/gh_branch_protect"
	"github.com/mindersec/minder/internal/engine/actions/remediate/noop"
	"github.com/mindersec/minder/internal/engine/actions/remediate/pull_request"
	"github.com/mindersec/minder/internal/engine/actions/remediate/rest"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// ActionType is the type of the remediation engine
const ActionType engif.ActionType = "remediate"

// NewRuleRemediator creates a new rule remediator
func NewRuleRemediator(
	rt *pb.RuleType,
	provider provinfv1.Provider,
	setting models.ActionOpt,
) (engif.Action, error) {
	remediate := rt.Def.GetRemediate()
	if remediate == nil {
		return noop.NewNoopRemediate(ActionType)
	}

	// nolint:revive // let's keep the switch here, it would be nicer to extend a switch in the future
	switch remediate.GetType() {
	case rest.RemediateType:
		client, err := provinfv1.As[provinfv1.REST](provider)
		if err != nil {
			return nil, errors.New("provider does not implement rest trait")
		}
		if remediate.GetRest() == nil {
			return nil, fmt.Errorf("remediations engine missing rest configuration")
		}
		return rest.NewRestRemediate(ActionType, remediate.GetRest(), client, setting)

	case gh_branch_protect.RemediateType:
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			return nil, errors.New("provider does not implement git trait")
		}
		if remediate.GetGhBranchProtection() == nil {
			return nil, fmt.Errorf("remediations engine missing gh_branch_protection configuration")
		}
		return gh_branch_protect.NewGhBranchProtectRemediator(
			ActionType, remediate.GetGhBranchProtection(), client, setting)

	case pull_request.RemediateType:
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			return nil, errors.New("provider does not implement git trait")
		}
		if remediate.GetPullRequest() == nil {
			return nil, fmt.Errorf("remediations engine missing pull request configuration")
		}

		return pull_request.NewPullRequestRemediate(
			ActionType, remediate.GetPullRequest(), client, setting)
	}

	return nil, fmt.Errorf("unknown remediation type: %s", remediate.GetType())
}
