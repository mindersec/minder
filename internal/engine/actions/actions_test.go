// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/internal/engine/actions/remediate/pull_request"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	enginerr "github.com/mindersec/minder/pkg/engine/errors"
)

func TestShouldRemediate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prevStatus RemediationStatus
		hasPrev    bool
		evalErr    error
		expected   engif.ActionCmd
	}{
		// Happy path: eval success
		{
			name:       "eval success, prev success -> off",
			prevStatus: RemediationStatusSuccess,
			hasPrev:    true,
			evalErr:    nil,
			expected:   engif.ActionCmdOff,
		},
		{
			name:       "eval success, prev skipped -> do nothing",
			prevStatus: RemediationStatusSkipped,
			hasPrev:    true,
			evalErr:    nil,
			expected:   engif.ActionCmdDoNothing,
		},
		// Happy path: eval failure triggers remediation
		{
			name:       "eval failure, prev skipped -> on",
			prevStatus: RemediationStatusSkipped,
			hasPrev:    true,
			evalErr:    enginerr.NewErrEvaluationFailed("failed"),
			expected:   engif.ActionCmdOn,
		},
		{
			name:       "eval failure, prev success -> do nothing",
			prevStatus: RemediationStatusSuccess,
			hasPrev:    true,
			evalErr:    enginerr.NewErrEvaluationFailed("failed"),
			expected:   engif.ActionCmdDoNothing,
		},
		// Expected errors: eval error cases
		{
			name:       "eval error, prev skipped -> do nothing",
			prevStatus: RemediationStatusSkipped,
			hasPrev:    true,
			evalErr:    errors.New("random error"),
			expected:   engif.ActionCmdDoNothing,
		},
		{
			// NOTE: EvalStatusTypesError has an empty case body in shouldRemediate,
			// so eval errors fall through to the default DoNothing. This may be a bug
			// (see the comment on cases Error/Success in shouldRemediate).
			name:       "eval error, prev success -> do nothing",
			prevStatus: RemediationStatusSuccess,
			hasPrev:    true,
			evalErr:    errors.New("random error"),
			expected:   engif.ActionCmdDoNothing,
		},
		{
			name:       "eval failure, prev error -> do nothing",
			prevStatus: RemediationStatusError,
			hasPrev:    true,
			evalErr:    enginerr.NewErrEvaluationFailed("failed"),
			expected:   engif.ActionCmdDoNothing,
		},
		{
			name:       "eval error, prev error -> do nothing",
			prevStatus: RemediationStatusError,
			hasPrev:    true,
			evalErr:    errors.New("random error"),
			expected:   engif.ActionCmdDoNothing,
		},
		// Edge cases
		{
			name:       "eval skipped -> do nothing",
			prevStatus: RemediationStatusSkipped,
			hasPrev:    true,
			evalErr:    enginerr.ErrEvaluationSkipSilently,
			expected:   engif.ActionCmdDoNothing,
		},
		{
			name:     "no previous eval, eval failure -> on",
			hasPrev:  false,
			evalErr:  enginerr.NewErrEvaluationFailed("failed"),
			expected: engif.ActionCmdOn,
		},
		{
			name:     "no previous eval, eval success -> do nothing",
			hasPrev:  false,
			evalErr:  nil,
			expected: engif.ActionCmdDoNothing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var prev *PreviousEval
			if tt.hasPrev {
				prev = &PreviousEval{RemediationStatus: tt.prevStatus}
			}
			status := mapEvalStatus(tt.evalErr)
			got := shouldRemediate(prev, status)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestShouldAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		prevAlert AlertStatus
		hasPrev   bool
		evalErr   error
		remErr    error
		remType   string
		expected  engif.ActionCmd
	}{
		// Happy path: eval success
		{
			name:      "eval success, alert on -> off",
			prevAlert: AlertStatusOn,
			hasPrev:   true,
			evalErr:   nil,
			remErr:    nil,
			// Using pull_request.RemediateType so we reach the switch instead of
			// hitting the early-return in Case 1 (non-PR successful remediation).
			remType:  pull_request.RemediateType,
			expected: engif.ActionCmdOff,
		},
		{
			name:      "eval success, alert already off -> do nothing",
			prevAlert: AlertStatusOff,
			hasPrev:   true,
			evalErr:   nil,
			remErr:    nil,
			remType:   pull_request.RemediateType,
			expected:  engif.ActionCmdDoNothing,
		},
		// Happy path: eval failure triggers alert
		{
			name:      "pr remediation eval failure, alert skipped -> on",
			prevAlert: AlertStatusSkipped,
			hasPrev:   true,
			evalErr:   enginerr.NewErrEvaluationFailed("failed"),
			remErr:    nil,
			remType:   pull_request.RemediateType,
			expected:  engif.ActionCmdOn,
		},
		{
			name:      "pr remediation eval failure, alert already on -> do nothing",
			prevAlert: AlertStatusOn,
			hasPrev:   true,
			evalErr:   enginerr.NewErrEvaluationFailed("failed"),
			remErr:    nil,
			remType:   pull_request.RemediateType,
			expected:  engif.ActionCmdDoNothing,
		},
		// Non-PR successful remediation (Case 1 early return)
		{
			name:      "successful non-pr remediation, alert on -> off",
			prevAlert: AlertStatusOn,
			hasPrev:   true,
			evalErr:   enginerr.NewErrEvaluationFailed("failed"),
			remErr:    nil,
			remType:   "some-other-type",
			expected:  engif.ActionCmdOff,
		},
		{
			name:      "successful non-pr remediation, alert already off -> do nothing",
			prevAlert: AlertStatusOff,
			hasPrev:   true,
			evalErr:   enginerr.NewErrEvaluationFailed("failed"),
			remErr:    nil,
			remType:   "some-other-type",
			expected:  engif.ActionCmdDoNothing,
		},
		// Expected errors
		{
			name:      "eval error -> do nothing",
			prevAlert: AlertStatusOff,
			hasPrev:   true,
			evalErr:   errors.New("generic error"),
			remErr:    nil,
			remType:   pull_request.RemediateType,
			expected:  engif.ActionCmdDoNothing,
		},
		// Edge cases
		{
			name:     "no previous eval, eval failure -> on",
			hasPrev:  false,
			evalErr:  enginerr.NewErrEvaluationFailed("failed"),
			remErr:   nil,
			remType:  pull_request.RemediateType,
			expected: engif.ActionCmdOn,
		},
		{
			name:     "no previous eval, eval success -> off",
			hasPrev:  false,
			evalErr:  nil,
			remErr:   nil,
			remType:  pull_request.RemediateType,
			expected: engif.ActionCmdOff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var prev *PreviousEval
			if tt.hasPrev {
				prev = &PreviousEval{AlertStatus: tt.prevAlert}
			}
			status := mapEvalStatus(tt.evalErr)
			got := shouldAlert(prev, status, tt.remErr, tt.remType)
			assert.Equal(t, tt.expected, got)
		})
	}
}
