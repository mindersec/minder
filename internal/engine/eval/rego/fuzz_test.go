// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego

import (
	"context"
	"testing"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
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
		e.Eval(context.Background(), emptyPol, nil, &interfaces.Result{
			Object: map[string]any{
				"data": data,
			},
		})
	})
}
