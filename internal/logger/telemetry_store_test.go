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
	"time"

	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/logger"
)

type testData struct {
	Data   string `json:"data"`
	Number int    `json:"number"`
}

func (t *testData) MarshalJSON() ([]byte, error) {
	// N.B. you can't use json.Marshal(t), because that will recurse infinitely
	return json.Marshal(map[string]any{
		"data":   t.Data,
		"number": t.Number,
	})
}

func TestTelemetryStore_Record(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		telemetry *logger.TelemetryStore
		expected  string
	}{{
		name: "nil telemetry",
	}, {
		name:      "standard telemetry",
		telemetry: &logger.TelemetryStore{},
		expected:  `{"foo":"bar","foo2":{"foo":"bar"}}`,
	}, {
		name:      "typed recording telemetry",
		telemetry: &logger.TelemetryStore{},
		expected:  `{"duration": 3000, "count": 200, "complex": {"data": "test", "number": 1}}`,
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

			// This would normally be inside a function
			logger.BusinessRecord(ctx)["foo"] = "bar"
			data := struct {
				Foo string `json:"foo"`
			}{"bar"}
			logger.BusinessRecord(ctx)["foo2"] = data
			if err := logger.BusinessRecord2(ctx, "duration", time.Second*3); err != nil {
				t.Fatal("Unable to record duration:", err)
			}
			if err := logger.BusinessRecord2(ctx, "count", 200); err != nil {
				t.Fatal("Unable to record count:", err)
			}
			if err := logger.BusinessRecord3(ctx, "complex", &testData{Data: "test", Number: 1}); err != nil {
				t.Fatal("Unable to record complex data:", err)
			}

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
		})

	}
}
