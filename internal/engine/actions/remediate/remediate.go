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
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/stacklok/mediator/internal/engine/actions/remediate/noop"
	"github.com/stacklok/mediator/internal/engine/actions/remediate/rest"
	enginerr "github.com/stacklok/mediator/internal/engine/errors"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// ActionType is the type of the remediation engine
var ActionType engif.ActionType = "remediate"

// NewRuleRemediator creates a new rule remediator
func NewRuleRemediator(rt *pb.RuleType, pbuild *providers.ProviderBuilder) (engif.Action, error) {
	rem := rt.Def.GetRemediate()
	if rem == nil {
		return noop.NewNoopRemediate(ActionType, defaultIsSkippable)
	}

	// nolint:revive // let's keep the switch here, it would be nicer to extend a switch in the future
	switch rem.GetType() {
	case rest.RemediateType:
		if rem.GetRest() == nil {
			return nil, fmt.Errorf("remediations engine missing rest configuration")
		}
		return rest.NewRestRemediate(ActionType, defaultIsSkippable, rem.GetRest(), pbuild)
	}

	return nil, fmt.Errorf("unknown remediation type: %s", rem.GetType())
}

// defaultIsSkippable returns true if the remediation is skippable.
// This function can be used for various remediation types that have similar trigger conditions
// In case there's a type-specific behavior needed, a custom function can be passed.
func defaultIsSkippable(ctx context.Context, remAction engif.ActionOpt, evalErr error) bool {
	logger := zerolog.Ctx(ctx)
	var skipRemediation bool

	switch remAction {
	case engif.ActionOptOff:
		// Remediation is off, skip
		return true
	case engif.ActionOptUnknown:
		// Remediation is unknown, skip
		logger.Debug().Msg("unknown remediation action, check your profile definition")
		return true
	case engif.ActionOptDryRun, engif.ActionOptOn:
		// Remediation is on or dry-run, do not skip yet. Check the evaluation error
		skipRemediation =
			// rule evaluation was skipped, skip remediation
			errors.Is(evalErr, enginerr.ErrEvaluationSkipped) ||
				// rule evaluation was skipped silently, skip remediation
				errors.Is(evalErr, enginerr.ErrEvaluationSkipSilently) ||
				// rule evaluation had no error, skip remediation
				evalErr == nil
	}
	// everything else, do not skip
	return skipRemediation
}
