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

		//nolint:gosec // Ignore the error; The test does not need it
		e.Eval(context.Background(), emptyPol, &engif.Result{
			Object: map[string]any{
				"data": data,
			},
		})
	})
}
