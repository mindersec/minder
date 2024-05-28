// Copyright 2024 Stacklok, Inc.
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

package rego

import (
	"context"
	"testing"

	engif "github.com/stacklok/minder/internal/engine/interfaces"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// FuzzRegoEval tests for unexpected behavior in e.Eval().
// The test does not validate the return values from e.Eval().
func FuzzRegoEval(f *testing.F) {
	f.Fuzz(func(_ *testing.T, policy, data string) {
		e, err := NewRegoEvaluator(
			&minderv1.RuleType_Definition_Eval_Rego{
				Type: ConstraintsEvaluationType.String(),
				Def:  policy,
			},
		)
		if err != nil {
			return
		}

		emptyPol := map[string]any{}

		// Call the main target of this test. Ignore the return values;
		// The fuzzer tests for unexpected behavior, so it is not
		// important what e.Eval() returns. 
		//nolint:gosec // Ignore the return values from e.Eval()
		e.Eval(context.Background(), emptyPol, &engif.Result{
			Object: map[string]any{
				"data": data,
			},
		})
	})
}
