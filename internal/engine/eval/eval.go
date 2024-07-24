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

// Package eval provides necessary interfaces and implementations for evaluating
// rules.
package eval

import (
	"context"
	"errors"
	"fmt"

	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/application"
	"github.com/stacklok/minder/internal/engine/eval/jq"
	"github.com/stacklok/minder/internal/engine/eval/rego"
	"github.com/stacklok/minder/internal/engine/eval/trusty"
	"github.com/stacklok/minder/internal/engine/eval/vulncheck"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// NewRuleEvaluator creates a new rule data evaluator
func NewRuleEvaluator(
	ctx context.Context,
	ruletype *pb.RuleType,
	provider provinfv1.Provider,
) (engif.Evaluator, error) {
	e := ruletype.Def.GetEval()
	if e == nil {
		return nil, fmt.Errorf("rule type missing eval configuration")
	}

	// TODO: make this more generic and/or use constants
	switch ruletype.Def.Eval.Type {
	case "jq":
		if ruletype.Def.Eval.GetJq() == nil {
			return nil, fmt.Errorf("rule type engine missing jq configuration")
		}
		return jq.NewJQEvaluator(e.GetJq())
	case rego.RegoEvalType:
		return rego.NewRegoEvaluator(e.GetRego())
	case vulncheck.VulncheckEvalType:
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			return nil, errors.New("provider does not implement github trait")
		}
		return vulncheck.NewVulncheckEvaluator(client)
	case trusty.TrustyEvalType:
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			return nil, errors.New("provider does not implement github trait")
		}
		return trusty.NewTrustyEvaluator(ctx, client)
	case application.HomoglyphsEvalType:
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			return nil, errors.New("provider does not implement git trait")
		}
		return application.NewHomoglyphsEvaluator(e.GetHomoglyphs(), client)
	default:
		return nil, fmt.Errorf("unsupported rule type engine: %s", ruletype.Def.Eval.Type)
	}
}
