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
	eoptions "github.com/mindersec/minder/internal/engine/options"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
	"github.com/open-feature/go-sdk/openfeature"
)

// NewRuleEvaluator creates a new rule data evaluator
func NewRuleEvaluator(
	ctx context.Context,
	ruletype *minderv1.RuleType,
	provider provinfv1.Provider,
	featureFlags openfeature.IClient,
	opts ...eoptions.Option,
) (interfaces.Evaluator, error) {
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
		return jq.NewJQEvaluator(e.GetJq(), opts...)
	case rego.RegoEvalType:
		return rego.NewRegoEvaluator(e.GetRego(), featureFlags, opts...)
	case vulncheck.VulncheckEvalType:
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			return nil, errors.New("provider does not implement github trait")
		}
		return vulncheck.NewVulncheckEvaluator(client, opts...)
	case trusty.TrustyEvalType:
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			return nil, errors.New("provider does not implement github trait")
		}
		return trusty.NewTrustyEvaluator(ctx, client, opts...)
	case application.HomoglyphsEvalType:
		client, err := provinfv1.As[provinfv1.GitHub](provider)
		if err != nil {
			return nil, errors.New("provider does not implement git trait")
		}
		return application.NewHomoglyphsEvaluator(ctx, e.GetHomoglyphs(), client, opts...)
	default:
		return nil, fmt.Errorf("unsupported rule type engine: %s", ruletype.Def.Eval.Type)
	}
}
