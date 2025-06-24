// Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/internal/util/cli/table/layouts"
)

// testEvalStatus implements the EvalStatus interface for testing
type testEvalStatus struct {
	status            string
	statusDetail      string
	remediationStatus string
	remediationDetail string
	alert             *testStatusDetails
}

// testStatusDetails implements the StatusDetails interface for testing
type testStatusDetails struct {
	status  string
	details string
}

func (m *testStatusDetails) GetStatus() string {
	return m.status
}

func (m *testStatusDetails) GetDetails() string {
	return m.details
}

func (m *testEvalStatus) GetStatus() string {
	return m.status
}

func (m *testEvalStatus) GetStatusDetail() string {
	return m.statusDetail
}

func (m *testEvalStatus) GetRemediationStatus() string {
	return m.remediationStatus
}

func (m *testEvalStatus) GetRemediationDetail() string {
	return m.remediationDetail
}

func (m *testEvalStatus) GetAlert() StatusDetails {
	if m.alert == nil {
		return &testStatusDetails{status: "off", details: ""}
	}
	return m.alert
}

func TestGetStatusIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		evalStatus       *testEvalStatus
		expectedEmoji    string
		expectedText     string
		expectedColor    layouts.Color
		expectedSeverity int
	}{
		{
			name: "success status",
			evalStatus: &testEvalStatus{
				status: successStatus,
			},
			expectedEmoji:    "✅",
			expectedText:     "Ok",
			expectedColor:    layouts.ColorGreen,
			expectedSeverity: 1,
		},
		{
			name: "rule eval error",
			evalStatus: &testEvalStatus{
				status: errorStatus,
			},
			expectedEmoji:    "🚧",
			expectedText:     "Error",
			expectedColor:    layouts.ColorRed,
			expectedSeverity: 3,
		},
		{
			name: "rule eval skipped",
			evalStatus: &testEvalStatus{
				status: skippedStatus,
			},
			expectedEmoji:    "➖",
			expectedText:     "Skipped",
			expectedColor:    layouts.ColorYellow,
			expectedSeverity: 2,
		},
		{
			name: "successfully remediated",
			evalStatus: &testEvalStatus{
				status:            failureStatus,
				remediationStatus: successStatus,
			},
			expectedEmoji:    "🔧",
			expectedText:     "Fixed",
			expectedColor:    layouts.ColorGreen,
			expectedSeverity: 1,
		},
		{
			name: "failed remediation on violation",
			evalStatus: &testEvalStatus{
				status:            failureStatus,
				remediationStatus: failureStatus,
			},
			expectedEmoji:    "⛓️‍💥",
			expectedText:     "No fix",
			expectedColor:    layouts.ColorRed,
			expectedSeverity: 3,
		},
		{
			name: "remediation error",
			evalStatus: &testEvalStatus{
				status:            failureStatus,
				remediationStatus: errorStatus,
				alert:             &testStatusDetails{status: offStatus},
			},
			expectedEmoji:    "😷",
			expectedText:     "!Fix",
			expectedColor:    layouts.ColorRed,
			expectedSeverity: 3,
		},
		{
			name: "failure with alert",
			evalStatus: &testEvalStatus{
				status: failureStatus,
				alert:  &testStatusDetails{status: onStatus},
			},
			expectedEmoji:    "🚨",
			expectedText:     "Alert",
			expectedColor:    layouts.ColorYellow,
			expectedSeverity: 2,
		},
		{
			name: "error on alert",
			evalStatus: &testEvalStatus{
				status:            failureStatus,
				remediationStatus: skippedStatus,
				alert:             &testStatusDetails{status: errorStatus},
			},
			expectedEmoji:    "🤮",
			expectedText:     "!Alert",
			expectedColor:    layouts.ColorRed,
			expectedSeverity: 3,
		},
		{
			name: "failure with no actions",
			evalStatus: &testEvalStatus{
				status:            failureStatus,
				remediationStatus: notAvailableStatus,
				alert:             &testStatusDetails{status: offStatus},
			},
			expectedEmoji:    "⛔",
			expectedText:     "Failed",
			expectedColor:    layouts.ColorRed,
			expectedSeverity: 3,
		},
		{
			name: "failure with no actions (different status)",
			evalStatus: &testEvalStatus{
				status:            failureStatus,
				remediationStatus: skippedStatus,
				alert:             &testStatusDetails{status: notAvailableStatus},
			},
			expectedEmoji:    "⛔",
			expectedText:     "Failed",
			expectedColor:    layouts.ColorRed,
			expectedSeverity: 3,
		},
		{
			name: "unknown status with emoji",
			evalStatus: &testEvalStatus{
				status: "pants!",
			},
			expectedEmoji:    "❓",
			expectedText:     "Unknown",
			expectedColor:    "",
			expectedSeverity: 0,
		},
		{
			name: "fix and alert",
			evalStatus: &testEvalStatus{
				status:            failureStatus,
				remediationStatus: successStatus,
				alert:             &testStatusDetails{status: onStatus},
			},
			expectedEmoji:    "🔧🚨",
			expectedText:     "Fixed Alert",
			expectedColor:    layouts.ColorYellow,
			expectedSeverity: 2,
		},
		{
			name: "remediation error and alert error",
			evalStatus: &testEvalStatus{
				status:            failureStatus,
				remediationStatus: errorStatus,
				alert:             &testStatusDetails{status: errorStatus},
			},
			expectedEmoji:    "😷🤮",
			expectedText:     "!Fix !Alert",
			expectedColor:    layouts.ColorRed,
			expectedSeverity: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emojiResult := GetStatusIcon(tt.evalStatus, true)

			assert.Equal(t, tt.expectedEmoji, emojiResult.Column, "unexpected emoji output")
			assert.Equal(t, tt.expectedColor, emojiResult.Color, "unexpected color")

			textResult := GetStatusIcon(tt.evalStatus, false)
			assert.Equal(t, tt.expectedText, textResult.Column, "unexpected text output")
			assert.Equal(t, tt.expectedColor, textResult.Color, "unexpected color")
		})
	}
}

func TestGetStatusIcon_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil alert returns default off status", func(t *testing.T) {
		t.Parallel()

		evalStatus := &testEvalStatus{
			status: failureStatus,
			alert:  nil, // explicitly nil
		}

		result := GetStatusIcon(evalStatus, false)

		// Should default to "fail no fix" since no remediation and alert is effectively "off"
		assert.Equal(t, "Failed", result.Column)
		assert.Equal(t, layouts.ColorRed, result.Color)
	})

	t.Run("empty status string should be treated as unknown", func(t *testing.T) {
		t.Parallel()

		evalStatus := &testEvalStatus{
			status: "", // empty status
		}

		result := GetStatusIcon(evalStatus, true)

		assert.Equal(t, "❓", result.Column)
		assert.Equal(t, layouts.Color(""), result.Color) // no color for unknown
	})
}
