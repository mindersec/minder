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
	"fmt"
	"os"

	"github.com/stacklok/mediator/internal/engine/eval/jq"
	"github.com/stacklok/mediator/internal/engine/eval/package_intelligence"
	"github.com/stacklok/mediator/internal/engine/eval/rego"
	"github.com/stacklok/mediator/internal/engine/eval/vulncheck"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// NewRuleEvaluator creates a new rule data evaluator
func NewRuleEvaluator(rt *pb.RuleType, cli *providers.ProviderBuilder) (engif.Evaluator, error) {
	e := rt.Def.GetEval()
	if e == nil {
		return nil, fmt.Errorf("rule type missing eval configuration")
	}

	// TODO: make this more generic and/or use constants
	switch rt.Def.Eval.Type {
	case "jq":
		if rt.Def.Eval.GetJq() == nil {
			return nil, fmt.Errorf("rule type engine missing jq configuration")
		}

		return jq.NewJQEvaluator(e.GetJq())
	case rego.RegoEvalType:
		return rego.NewRegoEvaluator(e.GetRego())
	case vulncheck.VulncheckEvalType:
		return vulncheck.NewVulncheckEvaluator(e.GetVulncheck(), cli)
	case package_intelligence.PiEvalType:
		pie := e.GetPackageIntelligence()
		if pie == nil {
			return nil, fmt.Errorf("rule type engine missing package_intelligence configuration")
		}
		if pie.GetEndpoint() == "" {
			pie.Endpoint = os.Getenv("MEDIATOR_UNSTABLE_PACKAGE_INTELLIGENCE_ENDPOINT")
		}
		return package_intelligence.NewPackageIntelligenceEvaluator(pie, cli)
	default:
		return nil, fmt.Errorf("unsupported rule type engine: %s", rt.Def.Eval.Type)
	}
}
