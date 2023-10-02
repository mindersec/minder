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

	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/engine/remediate/noop"
	"github.com/stacklok/mediator/internal/engine/remediate/rest"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// NewRuleRemediator creates a new rule remediator
func NewRuleRemediator(rt *pb.RuleType, pbuild *providers.ProviderBuilder) (engif.Remediator, error) {
	rem := rt.Def.GetRemediate()
	if rem == nil {
		return noop.NewNoopRemediate()
	}

	// nolint:revive // let's keep the switch here, it would be nicer to extend a switch in the future
	switch rem.GetType() {
	case rest.RemediateType:
		if rem.GetRest() == nil {
			return nil, fmt.Errorf("remediations engine missing rest configuration")
		}

		return rest.NewRestRemediate(rem.GetRest(), pbuild)
	}

	return nil, fmt.Errorf("unknown remediation type: %s", rem.GetType())
}
