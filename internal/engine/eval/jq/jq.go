// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package jq provides the jq profile evaluator
package jq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/reflect/protoreflect"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// Evaluator is an Evaluator that uses the jq library to evaluate rules
type Evaluator struct {
	assertions []*pb.RuleType_Definition_Eval_JQComparison
}

// NewJQEvaluator creates a new JQ rule data evaluator
func NewJQEvaluator(
	assertions []*pb.RuleType_Definition_Eval_JQComparison,
	opts ...interfaces.Option,
) (*Evaluator, error) {
	if len(assertions) == 0 {
		return nil, fmt.Errorf("missing jq assertions")
	}

	for idx := range assertions {
		a := assertions[idx]
		if err := a.Validate(); err != nil {
			return nil, fmt.Errorf("invalid jq assertion: %w", err)
		}
	}

	evaluator := &Evaluator{
		assertions: assertions,
	}

	for _, opt := range opts {
		if err := opt(evaluator); err != nil {
			return nil, err
		}
	}

	return evaluator, nil
}

// Eval calls the jq library to evaluate the rule
func (jqe *Evaluator) Eval(
	ctx context.Context,
	pol map[string]any,
	_ protoreflect.ProtoMessage,
	res *interfaces.Ingested,
) (*interfaces.EvaluationResult, error) {
	if res.Object == nil {
		return nil, fmt.Errorf("missing object")
	}
	obj := res.Object

	for idx := range jqe.assertions {
		var expectedVal, dataVal any
		var err error

		a := jqe.assertions[idx]

		// If there is no profile accessor, get the expected value from the constant accessor
		if a.Profile == nil {
			expectedVal, err = util.JQReadConstant[any](a.Constant.AsInterface())
			if err != nil {
				return nil, fmt.Errorf("cannot get values from profile accessor: %w", err)
			}
		} else {
			// Get the expected value from the profile accessor
			expectedVal, err = util.JQReadFrom[any](ctx, a.Profile.Def, pol)
			// we ignore util.ErrNoValueFound because we want to allow the JQ accessor to return the default value
			// which is fine for DeepEqual
			if err != nil && !errors.Is(err, util.ErrNoValueFound) {
				return nil, fmt.Errorf("cannot get values from profile accessor: %w", err)
			}
		}

		dataVal, err = util.JQReadFrom[any](ctx, a.Ingested.Def, obj)
		if err != nil && !errors.Is(err, util.ErrNoValueFound) {
			return nil, fmt.Errorf("cannot get values from data accessor: %w", err)
		}

		// Deep compare
		if !reflect.DeepEqual(standardizeNumbers(expectedVal), standardizeNumbers(dataVal)) {
			msg := fmt.Sprintf("data does not match profile: for assertion %d, got %v, want %v",
				idx, dataVal, expectedVal)

			marshalledAssertion, err := json.MarshalIndent(a, "", "  ")
			if err == nil {
				msg = fmt.Sprintf("%s\nassertion: %s", msg, string(marshalledAssertion))
			}

			return nil, evalerrors.NewDetailedErrEvaluationFailed(
				templates.JqTemplate,
				map[string]any{
					"path":     a.Ingested.Def,
					"expected": expectedVal,
					"actual":   dataVal,
				},
				"%s",
				msg,
			)
		}
	}

	return &interfaces.EvaluationResult{}, nil
}

// Convert numeric types to float64
func standardizeNumbers(v any) any {
	switch v := v.(type) {
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return v
	}
}
