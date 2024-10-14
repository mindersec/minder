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
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	eoptions "github.com/mindersec/minder/internal/engine/options"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// Evaluator is an Evaluator that uses the jq library to evaluate rules
type Evaluator struct {
	assertions []*pb.RuleType_Definition_Eval_JQComparison
}

// NewJQEvaluator creates a new JQ rule data evaluator
func NewJQEvaluator(
	assertions []*pb.RuleType_Definition_Eval_JQComparison,
	opts ...eoptions.Option,
) (*Evaluator, error) {
	if len(assertions) == 0 {
		return nil, fmt.Errorf("missing jq assertions")
	}

	for idx := range assertions {
		a := assertions[idx]
		if a.Profile != nil && a.Constant != nil {
			return nil, fmt.Errorf("profile and constant accessors are mutually exclusive")
		} else if a.Profile == nil && a.Constant == nil {
			return nil, fmt.Errorf("missing profile or constant accessor")
		}

		if a.Profile != nil && a.Profile.Def == "" {
			return nil, fmt.Errorf("missing profile accessor definition")
		}

		if a.Ingested == nil {
			return nil, fmt.Errorf("missing data accessor")
		}

		if a.Ingested.Def == "" {
			return nil, fmt.Errorf("missing data accessor definition")
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
func (jqe *Evaluator) Eval(ctx context.Context, pol map[string]any, _ protoreflect.ProtoMessage, res *engif.Result) error {
	if res.Object == nil {
		return fmt.Errorf("missing object")
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
				return fmt.Errorf("cannot get values from profile accessor: %w", err)
			}
		} else {
			// Get the expected value from the profile accessor
			expectedVal, err = util.JQReadFrom[any](ctx, a.Profile.Def, pol)
			// we ignore util.ErrNoValueFound because we want to allow the JQ accessor to return the default value
			// which is fine for DeepEqual
			if err != nil && !errors.Is(err, util.ErrNoValueFound) {
				return fmt.Errorf("cannot get values from profile accessor: %w", err)
			}
		}

		dataVal, err = util.JQReadFrom[any](ctx, a.Ingested.Def, obj)
		if err != nil && !errors.Is(err, util.ErrNoValueFound) {
			return fmt.Errorf("cannot get values from data accessor: %w", err)
		}

		// Deep compare
		if !reflect.DeepEqual(standardizeNumbers(expectedVal), standardizeNumbers(dataVal)) {
			msg := fmt.Sprintf("data does not match profile: for assertion %d, got %v, want %v",
				idx, dataVal, expectedVal)

			marshalledAssertion, err := json.MarshalIndent(a, "", "  ")
			if err == nil {
				msg = fmt.Sprintf("%s\nassertion: %s", msg, string(marshalledAssertion))
			}

			return evalerrors.NewErrEvaluationFailed("%s", msg)
		}
	}

	return nil
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
