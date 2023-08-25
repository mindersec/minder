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
	"context"
	"errors"
	"fmt"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Evaluator is the interface for a rule type evaluator
type Evaluator interface {
	Eval(ctx context.Context, policy map[string]any, obj any) error
}

// ErrEvaluationFailed is an error that occurs during evaluation of a rule.
var ErrEvaluationFailed = errors.New("evaluation error")

// NewErrEvaluationFailed creates a new evaluation error
func NewErrEvaluationFailed(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", ErrEvaluationFailed, msg)
}

// NewRuleEvaluator creates a new rule data evaluator
func NewRuleEvaluator(rt *pb.RuleType) (Evaluator, error) {
	ing := rt.Def.GetEval()
	if ing == nil {
		return nil, fmt.Errorf("rule type missing eval configuration")
	}

	// TODO: make this more generic and/or use constants
	switch rt.Def.Eval.Type {
	case "jq":
		if rt.Def.Eval.GetJq() == nil {
			return nil, fmt.Errorf("rule type engine missing rest configuration")
		}

		return NewJQEvaluator(ing.GetJq())
	default:
		return nil, fmt.Errorf("unsupported rule type engine: %s", rt.Def.Eval.Type)
	}
}
