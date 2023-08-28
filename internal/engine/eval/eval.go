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
// Package rule provides the CLI subcommand for managing rules

// Package eval provides necessary interfaces and implementations for evaluating
// rules.
package eval

import (
	"fmt"

	"github.com/stacklok/mediator/internal/engine/eval/jq"
	"github.com/stacklok/mediator/internal/engine/eval/rego"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// NewRuleEvaluator creates a new rule data evaluator
func NewRuleEvaluator(rt *pb.RuleType) (engif.Evaluator, error) {
	e := rt.Def.GetEval()
	if e == nil {
		return nil, fmt.Errorf("rule type missing eval configuration")
	}

	// TODO: make this more generic and/or use constants
	switch rt.Def.Eval.Type {
	case "jq":
		if rt.Def.Eval.GetJq() == nil {
			return nil, fmt.Errorf("rule type engine missing rest configuration")
		}

		return jq.NewJQEvaluator(e.GetJq())
	case rego.RegoEvalType:
		return rego.NewRegoEvaluator(e.GetRego())
	default:
		return nil, fmt.Errorf("unsupported rule type engine: %s", rt.Def.Eval.Type)
	}
}
