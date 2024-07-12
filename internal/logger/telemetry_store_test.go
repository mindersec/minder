// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/actions/alert"
	"github.com/stacklok/minder/internal/engine/actions/remediate"
	enginerr "github.com/stacklok/minder/internal/engine/errors"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/logger"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestTelemetryStore_Record(t *testing.T) {
	testUUIDString := "00000000-0000-0000-0000-000000000001"
	testUUID := uuid.MustParse(testUUIDString)

	t.Parallel()
	cases := []struct {
		name           string
		telemetry      *logger.TelemetryStore
		evalParamsFunc func() *engif.EvalStatusParams
		recordFunc     func(context.Context, engif.ActionsParams)
		expected       string
		notPresent     []string
	}{{
		name: "nil telemetry",
		evalParamsFunc: func() *engif.EvalStatusParams {
			ep := &engif.EvalStatusParams{}

			ep.Profile = &minderv1.Profile{
				Name: "artifact_profile",
				Id:   &testUUIDString,
			}
			ep.RuleTypeName = "artifact_signature"
			ep.SetEvalErr(enginerr.NewErrEvaluationFailed("evaluation failure reason"))
			ep.SetActionsOnOff(map[engif.ActionType]engif.ActionOpt{
				alert.ActionType:     engif.ActionOptOn,
				remediate.ActionType: engif.ActionOptOff,
			})
			ep.SetActionsErr(context.Background(), enginerr.ActionsError{
				RemediateErr: nil,
				AlertErr:     enginerr.ErrActionSkipped,
			})
			return ep
		},
		recordFunc: func(ctx context.Context, evalParams engif.ActionsParams) {
			logger.BusinessRecord(ctx).Project = testUUID
			logger.BusinessRecord(ctx).Repository = testUUID
			logger.BusinessRecord(ctx).AddRuleEval(evalParams)
		},
	}, {
		name:      "standard telemetry",
		telemetry: &logger.TelemetryStore{},
		evalParamsFunc: func() *engif.EvalStatusParams {
			ep := &engif.EvalStatusParams{}

			ep.Profile = &minderv1.Profile{
				Name: "artifact_profile",
				Id:   &testUUIDString,
			}
			ep.RuleTypeName = "artifact_signature"
			ep.RuleTypeID = testUUID
			ep.SetEvalErr(enginerr.NewErrEvaluationFailed("evaluation failure reason"))
			ep.SetActionsOnOff(map[engif.ActionType]engif.ActionOpt{
				alert.ActionType:     engif.ActionOptOff,
				remediate.ActionType: engif.ActionOptOn,
			})
			ep.SetActionsErr(context.Background(), enginerr.ActionsError{
				RemediateErr: nil,
				AlertErr:     enginerr.ErrActionSkipped,
			})
			return ep
		},
		recordFunc: func(ctx context.Context, evalParams engif.ActionsParams) {
			logger.BusinessRecord(ctx).Project = testUUID
			logger.BusinessRecord(ctx).Repository = testUUID
			logger.BusinessRecord(ctx).AddRuleEval(evalParams)
		},
		expected: `{
    "project": "00000000-0000-0000-0000-000000000001",
    "repository": "00000000-0000-0000-0000-000000000001",
    "rules": [
        {
			"ruletype": {
				"name": "artifact_signature",
				"id": "00000000-0000-0000-0000-000000000001"
			},
			"profile": {
				"name": "artifact_profile",
				"id": "00000000-0000-0000-0000-000000000001"
			},
			"eval_result": "failure",
			"actions": {
				"alert": {
					"state": "off",
					"result": "skipped"
				},
				"remediate": {
					"state": "on",
					"result": "success"
				}
			}
        }
    ]
    }`,
	}, {
		name:      "empty telemetry",
		telemetry: &logger.TelemetryStore{},
		evalParamsFunc: func() *engif.EvalStatusParams {
			return nil
		},
		recordFunc: func(_ context.Context, _ engif.ActionsParams) {
		},
		expected:   `{"telemetry": "true"}`,
		notPresent: []string{"project", "rules", "login_sha", "repository", "provider", "profile", "ruletypes", "artifact", "pr"},
	}}

	count := len(cases)
	t.Log("Running", count, "test cases")

	for _, testcase := range cases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			logBuf := bytes.Buffer{}
			zlog := zerolog.New(&logBuf)

			// Create a new TelemetryStore instance
			ctx := tc.telemetry.WithTelemetry(context.Background())

			tc.recordFunc(ctx, tc.evalParamsFunc())

			tc.telemetry.Record(zlog.Info()).Send()

			if tc.expected == "" {
				return
			}

			expected := map[string]any{}
			if err := json.Unmarshal([]byte(tc.expected), &expected); err != nil {
				t.Fatal("Unable to unmarshal expected data:", err)
			}

			got := map[string]any{}
			if err := json.Unmarshal(logBuf.Bytes(), &got); err != nil {
				t.Fatal("Unable to unmarshal log data:", err)
			}

			for key, value := range expected {
				if !reflect.DeepEqual(value, got[key]) {
					t.Errorf("Expected %s for %q, got %s", value, key, got[key])
				}
			}

			for _, key := range tc.notPresent {
				if _, ok := got[key]; ok {
					t.Errorf("Expected %q to not be present in %s, but it was", key, got)
				}
			}
		})

	}
}
