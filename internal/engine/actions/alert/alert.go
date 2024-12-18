// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package alert provides necessary interfaces and implementations for
// processing alerts.
package alert

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/engine/actions/alert/noop"
	"github.com/mindersec/minder/internal/engine/actions/alert/pull_request_comment"
	"github.com/mindersec/minder/internal/engine/actions/alert/security_advisory"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// ActionType is the type of the alert engine
const ActionType engif.ActionType = "alert"

// NewRuleAlert creates a new rule alert engine
func NewRuleAlert(
	ctx context.Context,
	ruletype *pb.RuleType,
	provider provinfv1.Provider,
	setting models.ActionOpt,
) (engif.Action, error) {
	alertCfg := ruletype.Def.GetAlert()
	if alertCfg == nil {
		return noop.NewNoopAlert(ActionType)
	}

	// nolint:revive // let's keep the switch here, it would be nicer to extend a switch in the future
	switch alertCfg.GetType() {
	case security_advisory.AlertType:
		if alertCfg.GetSecurityAdvisory() == nil {
			return nil, fmt.Errorf("alert engine missing security-advisory configuration")
		}
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Str("rule-type", ruletype.GetName()).
				Msg("provider is not a GitHub provider. Silently skipping alerts.")
			return noop.NewNoopAlert(ActionType)
		}
		return security_advisory.NewSecurityAdvisoryAlert(
			ActionType, ruletype, alertCfg.GetSecurityAdvisory(), client, setting)
	case pull_request_comment.AlertType:
		if alertCfg.GetPullRequestComment() == nil {
			return nil, fmt.Errorf("alert engine missing pull_request_review configuration")
		}
		client, err := provinfv1.As[provinfv1.PullRequestCommenter](provider)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Str("rule-type", ruletype.GetName()).
				Msg("provider is not a GitHub provider. Silently skipping alerts.")
			return noop.NewNoopAlert(ActionType)
		}
		return pull_request_comment.NewPullRequestCommentAlert(
			ActionType, alertCfg.GetPullRequestComment(), client, setting,
			defaultName(ruletype))
	}

	return nil, fmt.Errorf("unknown alert type: %s", alertCfg.GetType())
}

func defaultName(ruletype *pb.RuleType) string {
	if ruletype.GetDisplayName() != "" {
		return ruletype.GetDisplayName()
	}
	return ruletype.GetName()
}
