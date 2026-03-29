// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/actions/remediate/pull_request"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
)

func TestShouldRemediate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow
		evalErr        error
		expected       engif.ActionCmd
	}{
		{
			name:           "eval success, prev skipped -> off",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{RemStatus: db.RemediationStatusTypesSuccess},
			evalErr:        nil, // success
			expected:       engif.ActionCmdOff,
		},
		{
			name:           "eval success, prev skipped -> do nothing",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{RemStatus: db.RemediationStatusTypesSkipped},
			evalErr:        nil,
			expected:       engif.ActionCmdDoNothing,
		},
		{
			name:           "eval failure, prev skipped -> on",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{RemStatus: db.RemediationStatusTypesSkipped},
			evalErr:        enginerr.NewErrEvaluationFailed("failed"),
			expected:       engif.ActionCmdOn,
		},
		{
			name:           "eval failure, prev success -> do nothing",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{RemStatus: db.RemediationStatusTypesSuccess},
			evalErr:        enginerr.NewErrEvaluationFailed("failed"),
			expected:       engif.ActionCmdDoNothing,
		},
		{
			name:           "eval error, prev skipped -> do nothing",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{RemStatus: db.RemediationStatusTypesSkipped},
			evalErr:        errors.New("random error"),
			expected:       engif.ActionCmdDoNothing,
		},
		{
			name:           "eval skipped -> do nothing",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{RemStatus: db.RemediationStatusTypesSkipped},
			evalErr:        enginerr.ErrEvaluationSkipSilently,
			expected:       engif.ActionCmdDoNothing,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldRemediate(tt.prevEvalFromDb, tt.evalErr)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestShouldAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow
		evalErr        error
		remErr         error
		remType        string
		expected       engif.ActionCmd
	}{
		{
			name:           "successful non-pr remediation with alert not off",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{AlertStatus: db.AlertStatusTypesOn},
			evalErr:        enginerr.NewErrEvaluationFailed("failed"),
			remErr:         nil,
			remType:        "some-other-type",
			expected:       engif.ActionCmdOff,
		},
		{
			name:           "successful non-pr remediation with alert already off",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{AlertStatus: db.AlertStatusTypesOff},
			evalErr:        enginerr.NewErrEvaluationFailed("failed"),
			remErr:         nil,
			remType:        "some-other-type",
			expected:       engif.ActionCmdDoNothing,
		},
		{
			name:           "pr remediation eval failure and alert skipped -> on",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{AlertStatus: db.AlertStatusTypesSkipped},
			evalErr:        enginerr.NewErrEvaluationFailed("failed"),
			remErr:         nil,
			remType:        pull_request.RemediateType,
			expected:       engif.ActionCmdOn,
		},
		{
			name:           "pr remediation eval failure and alert already on -> do nothing",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{AlertStatus: db.AlertStatusTypesOn},
			evalErr:        enginerr.NewErrEvaluationFailed("failed"),
			remErr:         nil,
			remType:        pull_request.RemediateType,
			expected:       engif.ActionCmdDoNothing,
		},
		{
			name:           "eval success and alert on -> off",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{AlertStatus: db.AlertStatusTypesOn},
			evalErr:        nil,
			remErr:         nil,
			remType:        pull_request.RemediateType,
			expected:       engif.ActionCmdOff,
		},
		{
			name:           "eval success and alert already off -> do nothing",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{AlertStatus: db.AlertStatusTypesOff},
			evalErr:        nil,
			remErr:         nil,
			remType:        pull_request.RemediateType,
			expected:       engif.ActionCmdDoNothing,
		},
		{
			name:           "eval error -> do nothing",
			prevEvalFromDb: &db.ListRuleEvaluationsByProfileIdRow{AlertStatus: db.AlertStatusTypesOff},
			evalErr:        errors.New("generic error"),
			remErr:         nil,
			remType:        pull_request.RemediateType,
			expected:       engif.ActionCmdDoNothing,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldAlert(tt.prevEvalFromDb, tt.evalErr, tt.remErr, tt.remType)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetDefaultResult(t *testing.T) {
	t.Parallel()

	res := getDefaultResult(context.Background())
	assert.ErrorIs(t, res.RemediateErr, enginerr.ErrActionSkipped)
	assert.ErrorIs(t, res.AlertErr, enginerr.ErrActionSkipped)

	var remMeta map[string]any
	require.NoError(t, json.Unmarshal(res.RemediateMeta, &remMeta))
	assert.Empty(t, remMeta)

	var alertMeta map[string]any
	require.NoError(t, json.Unmarshal(res.AlertMeta, &alertMeta))
	assert.Empty(t, alertMeta)
}

func TestGetRemediationMeta(t *testing.T) {
	t.Parallel()
	t.Run("nil profile", func(t *testing.T) {
		assert.Nil(t, getRemediationMeta(nil))
	})
	t.Run("valid profile", func(t *testing.T) {
		meta := json.RawMessage(`{"key": "value"}`)
		got := getRemediationMeta(&db.ListRuleEvaluationsByProfileIdRow{RemMetadata: meta})
		require.NotNil(t, got)
		assert.Equal(t, &meta, got)
	})
}

func TestGetAlertMeta(t *testing.T) {
	t.Parallel()
	t.Run("nil profile", func(t *testing.T) {
		assert.Nil(t, getAlertMeta(nil))
	})
	t.Run("valid profile", func(t *testing.T) {
		meta := json.RawMessage(`{"alert": "yes"}`)
		got := getAlertMeta(&db.ListRuleEvaluationsByProfileIdRow{AlertMetadata: meta})
		require.NotNil(t, got)
		assert.Equal(t, &meta, got)
	})
}
