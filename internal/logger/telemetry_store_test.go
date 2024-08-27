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
	"github.com/stacklok/minder/internal/profiles/models"
)

func TestTelemetryStore_Record(t *testing.T) {
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

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
			ep.Rule = &models.RuleInstance{
				RuleTypeID: testUUID,
			}
			ep.Profile = &models.ProfileAggregate{
				Name: "artifact_profile",
				ID:   testUUID,
			}
			ep.SetEvalErr(enginerr.NewErrEvaluationFailed("evaluation failure reason"))
			ep.SetActionsOnOff(map[engif.ActionType]models.ActionOpt{
				alert.ActionType:     models.ActionOptOn,
				remediate.ActionType: models.ActionOptOff,
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
			logger.BusinessRecord(ctx).AddRuleEval(evalParams, ruleTypeName)
		},
	}, {
		name:      "standard telemetry",
		telemetry: &logger.TelemetryStore{},
		evalParamsFunc: func() *engif.EvalStatusParams {
			ep := &engif.EvalStatusParams{}
			ep.Rule = &models.RuleInstance{
				RuleTypeID: testUUID,
			}
			ep.Profile = &models.ProfileAggregate{
				Name: "artifact_profile",
				ID:   testUUID,
			}
			ep.SetEvalErr(enginerr.NewErrEvaluationFailed("evaluation failure reason"))
			ep.SetActionsOnOff(map[engif.ActionType]models.ActionOpt{
				alert.ActionType:     models.ActionOptOff,
				remediate.ActionType: models.ActionOptOn,
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
			logger.BusinessRecord(ctx).AddRuleEval(evalParams, ruleTypeName)
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

const ruleTypeName = "artifact_signature"

func TestProjectTombstoneEquals(t *testing.T) {
	t.Parallel()

	testUUID := uuid.New()

	cases := []struct {
		name     string
		pt1      logger.ProjectTombstone
		pt2      logger.ProjectTombstone
		expected bool
	}{
		{
			name:     "empty ProjectTombstone structs",
			pt1:      logger.ProjectTombstone{},
			pt2:      logger.ProjectTombstone{},
			expected: true,
		},
		{
			name: "ProjectTombstone structs with the same values",
			pt1: logger.ProjectTombstone{
				Project:           testUUID,
				ProfileCount:      1,
				RepositoriesCount: 2,
				Entitlements:      []string{"entitlement1", "entitlement2"},
			},
			pt2: logger.ProjectTombstone{
				Project:           testUUID,
				ProfileCount:      1,
				RepositoriesCount: 2,
				Entitlements:      []string{"entitlement1", "entitlement2"},
			},
			expected: true,
		},
		{
			name: "ProjectTombstone structs with different project IDs",
			pt1: logger.ProjectTombstone{
				Project: testUUID,
			},
			pt2: logger.ProjectTombstone{
				Project: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			},
			expected: false,
		},
		{
			name: "ProjectTombstone structs with different entitlements",
			pt1: logger.ProjectTombstone{
				Project:      testUUID,
				Entitlements: []string{"entitlement1", "entitlement2"},
			},
			pt2: logger.ProjectTombstone{
				Project:      testUUID,
				Entitlements: []string{"entitlement2", "entitlement3"},
			},
			expected: false,
		},
	}

	t.Log("Running", len(cases), "test cases")

	for _, testcase := range cases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.pt1.Equals(tc.pt2) != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, tc.pt1.Equals(tc.pt2))
			}
		})
	}
}
