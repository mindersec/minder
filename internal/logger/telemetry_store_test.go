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

	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/logger"
)

func TestTelemetryStore_Record(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		telemetry  *logger.TelemetryStore
		recordFunc func(context.Context)
		expected   string
		notPresent []string
	}{{
		name: "nil telemetry",
		recordFunc: func(ctx context.Context) {
			logger.BusinessRecord(ctx).Project = "bar"

			logger.BusinessRecord(ctx).Resource = "foo/repo"
			logger.BusinessRecord(ctx).AddRuleEval("artifact_signature", interfaces.ActionOptDryRun)
		},
	}, {
		name:      "standard telemetry",
		telemetry: &logger.TelemetryStore{},
		recordFunc: func(ctx context.Context) {
			logger.BusinessRecord(ctx).Project = "bar"

			logger.BusinessRecord(ctx).Resource = "foo/repo"
			logger.BusinessRecord(ctx).AddRuleEval("artifact_signature", interfaces.ActionOptDryRun)
		},
		expected: `{"project":"bar","resource":"foo/repo","rules":[{"name":"artifact_signature","action":2}]}`,
	}, {
		name:      "empty telemetry",
		telemetry: &logger.TelemetryStore{},
		recordFunc: func(ctx context.Context) {
		},
		expected:   `{"telemetry": true}`,
		notPresent: []string{"project", "resource", "rules", "login_sha"},
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

			tc.recordFunc(ctx)

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
