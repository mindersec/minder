// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package eval provides necessary interfaces and implementations for evaluating
// rules.
package eval

import (
	"context"
	"errors"
	"fmt"

	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/application"
	"github.com/mindersec/minder/internal/engine/eval/jq"
	"github.com/mindersec/minder/internal/engine/eval/rego"
	"github.com/mindersec/minder/internal/engine/eval/trusty"
	"github.com/mindersec/minder/internal/engine/eval/vulncheck"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// NewRuleEvaluator creates a new rule data evaluator
func NewRuleEvaluator(
	ctx context.Context,
	ruletype *minderv1.RuleType,
	provider interfaces.Provider,
	opts ...interfaces.Option,
) (interfaces.Evaluator, error) {
	e := ruletype.Def.GetEval()
	if e == nil {
		return nil, fmt.Errorf("rule type missing eval configuration")
	}

	// TODO: make this more generic and/or use constants
	// Note that the JQ and Rego evaluators get the data through ingestion.
	switch ruletype.Def.Eval.Type {
	case "jq":
		if ruletype.Def.Eval.GetJq() == nil {
			return nil, fmt.Errorf("rule type engine missing jq configuration")
		}
		return jq.NewJQEvaluator(e.GetJq(), opts...)
	case rego.RegoEvalType:
		return rego.NewRegoEvaluator(e.GetRego(), opts...)
	case vulncheck.VulncheckEvalType:
		client, err := interfaces.As[vulncheck.GitHubRESTAndPRClient](provider)
		if err != nil {
			return nil, errors.New("provider does not implement github trait")
		}
		return vulncheck.NewVulncheckEvaluator(client, opts...)
	case trusty.TrustyEvalType:
		client, err := interfaces.As[interfaces.GitHubIssuePRClient](provider)
		if err != nil {
			return nil, errors.New("provider does not implement github trait")
		}
		return trusty.NewTrustyEvaluator(ctx, client, opts...)
	case application.HomoglyphsEvalType:
		client, err := interfaces.As[interfaces.GitHubIssuePRClient](provider)
		if err != nil {
			return nil, errors.New("provider does not implement git trait")
		}
		return application.NewHomoglyphsEvaluator(ctx, e.GetHomoglyphs(), client, opts...)
	default:
		return nil, fmt.Errorf("unsupported rule type engine: %s", ruletype.Def.Eval.Type)
	}
}
