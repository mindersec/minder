// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package jq_test

import (
	"context"
	"testing"

	"github.com/mindersec/minder/internal/engine/eval/jq"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func FuzzJqEval(f *testing.F) {
	f.Fuzz(func(_ *testing.T, comp1, comp2, pol1, pol2, obj1, obj2 string) {
		assertions := []*pb.RuleType_Definition_Eval_JQComparison{
			{
				Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
					Def: comp1,
				},
				Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
					Def: comp2,
				},
			},
		}
		pol := map[string]any{
			pol1: pol2,
		}

		obj := map[string]any{
			obj1: obj2,
		}

		jqe, err := jq.NewJQEvaluator(assertions)
		if err != nil {
			return
		}

		//nolint:gosec // Do not validate the return values so ignore them
		jqe.Eval(context.Background(), pol, nil, &engif.Result{Object: obj})
	})
}
