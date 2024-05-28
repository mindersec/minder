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

package jq_test

import (
	"context"
	"testing"

	"github.com/stacklok/minder/internal/engine/eval/jq"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
		jqe.Eval(context.Background(), pol, &engif.Result{Object: obj})
	})
}
