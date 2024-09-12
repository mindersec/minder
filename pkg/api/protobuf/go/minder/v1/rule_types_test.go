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

package v1_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestSeverity_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       *minderv1.Severity
		want    []byte
		wantErr bool
	}{
		{
			name: "valid",
			s: &minderv1.Severity{
				Value: minderv1.Severity_VALUE_CRITICAL,
			},
			want: []byte(`{"value":"critical"}`),
		},
		{
			name: "unknown",
			s: &minderv1.Severity{
				Value: minderv1.Severity_VALUE_UNKNOWN,
			},
			want: []byte(`{"value":"unknown"}`),
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.s)
			if tt.wantErr {
				assert.Errorf(t, err, "Severity.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got, "expected %s, got %s", tt.want, got)
		})
	}
}

func TestSeverity_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		want    *minderv1.Severity
		wantErr bool
	}{
		{
			name: "valid",
			data: []byte(`{"value":"critical"}`),
			want: &minderv1.Severity{
				Value: minderv1.Severity_VALUE_CRITICAL,
			},
		},
		{
			name: "unknown",
			data: []byte(`{"value":"unknown"}`),
			want: &minderv1.Severity{
				Value: minderv1.Severity_VALUE_UNKNOWN,
			},
		},
		{
			name:    "invalid",
			data:    []byte(`{"value":"invalid"}`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &minderv1.Severity{}
			err := json.Unmarshal(tt.data, s)
			if tt.wantErr {
				assert.Errorf(t, err, "Severity.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, s, "expected %v, got %v", tt.want, s)
		})
	}
}

func TestRuleType_WithDefaultShortFailureMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    *minderv1.RuleType
		want *minderv1.RuleType
	}{
		{
			name: "nil RuleType",
			r:    nil,
			want: nil,
		},
		{
			name: "empty EvaluationFailureMessage",
			r:    &minderv1.RuleType{Name: "TestRule"},
			want: &minderv1.RuleType{ShortFailureMessage: "Rule TestRule evaluation failed", Name: "TestRule"},
		},
		{
			name: "non-empty EvaluationFailureMessage",
			r:    &minderv1.RuleType{ShortFailureMessage: "Custom message", Name: "TestRule"},
			want: &minderv1.RuleType{ShortFailureMessage: "Custom message", Name: "TestRule"},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.r.WithDefaultShortFailureMessage()
			assert.Equal(t, tt.want, got, "expected %v, got %v", tt.want, got)
		})
	}
}

func TestRuleTypeState_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       minderv1.RuleTypeReleasePhase
		want    []byte
		wantErr bool
	}{
		{
			name:    "valid-alpha",
			s:       minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_ALPHA,
			want:    []byte(`"alpha"`),
			wantErr: false,
		},
		{
			name:    "valid-beta",
			s:       minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_BETA,
			want:    []byte(`"beta"`),
			wantErr: false,
		},
		{
			name:    "valid-ga",
			s:       minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_GA,
			want:    []byte(`"ga"`),
			wantErr: false,
		},
		{
			name:    "valid-deprecated",
			s:       minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_DEPRECATED,
			want:    []byte(`"deprecated"`),
			wantErr: false,
		},
		{
			name:    "invalid",
			s:       99999,
			want:    []byte(`""`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(&tt.s)
			if tt.wantErr {
				assert.Errorf(t, err, "RuleTypeState.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got, "expected %s, got %s", tt.want, got)
		})
	}
}

func TestRuleTypeState_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		want    minderv1.RuleTypeReleasePhase
		wantErr bool
	}{
		{
			name:    "valid",
			data:    []byte(`"alpha"`),
			want:    minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_ALPHA,
			wantErr: false,
		},
		{
			name:    "wrong-state",
			data:    []byte(`"wrong-state"`),
			wantErr: true,
		},
		{
			name:    "invalid",
			data:    nil,
			want:    minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var s minderv1.RuleTypeReleasePhase
			err := json.Unmarshal(tt.data, &s)
			if tt.wantErr {
				assert.Errorf(t, err, "RuleTypeState.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, s, "expected %v, got %v", tt.want, s)
		})
	}
}
