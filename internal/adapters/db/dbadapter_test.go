// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dbadapter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
	engineerrors "github.com/mindersec/minder/pkg/engine/errors"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

func TestMapDbToEngine(t *testing.T) {
	t.Parallel()

	// Test nil row
	require.Nil(t, MapDbToEngine(nil), "Expected nil output for nil row")

	// Test valid row
	row := &db.ListRuleEvaluationsByProfileIdRow{
		EvalStatus:  db.EvalStatusTypesFailure,
		RemStatus:   db.RemediationStatusTypesSuccess,
		AlertStatus: db.AlertStatusTypesOn,
	}

	snap := MapDbToEngine(row)
	require.NotNil(t, snap)
	require.Equal(t, string(db.EvalStatusTypesFailure), string(snap.EvalStatus))
	require.Equal(t, string(db.RemediationStatusTypesSuccess), snap.RemediationStatus)
	require.Equal(t, string(db.AlertStatusTypesOn), snap.AlertStatus)
}

func TestErrorAsEvalStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected db.EvalStatusTypes
	}{
		{"nil error", nil, db.EvalStatusTypesSuccess},
		{"eval failed", interfaces.ErrEvaluationFailed, db.EvalStatusTypesFailure},
		{"eval skipped", interfaces.ErrEvaluationSkipped, db.EvalStatusTypesSkipped},
		{"other error", errors.New("some other error"), db.EvalStatusTypesError},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, ErrorAsEvalStatus(tc.err))
		})
	}
}

func TestErrorAsEvalDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"standard error", errors.New("standard error"), "standard error"},
		{"eval error without template", engineerrors.NewErrEvaluationFailed("basic failure"), "basic failure"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, ErrorAsEvalDetails(tc.err))
		})
	}
}

func TestErrorAsRemediationStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected db.RemediationStatusTypes
	}{
		{"nil error", nil, db.RemediationStatusTypesSuccess},
		{"action failed", engineerrors.ErrActionFailed, db.RemediationStatusTypesFailure},
		{"action skipped", engineerrors.ErrActionSkipped, db.RemediationStatusTypesSkipped},
		{"action not available", engineerrors.ErrActionNotAvailable, db.RemediationStatusTypesNotAvailable},
		{"action pending", engineerrors.ErrActionPending, db.RemediationStatusTypesPending},
		{"other error", errors.New("other error"), db.RemediationStatusTypesError},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, ErrorAsRemediationStatus(tc.err))
		})
	}
}

func TestRemediationStatusAsError(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, RemediationStatusAsError(nil), engineerrors.ErrActionSkipped)

	tests := []struct {
		name     string
		status   db.RemediationStatusTypes
		expected error
	}{
		{"success", db.RemediationStatusTypesSuccess, nil},
		{"failure", db.RemediationStatusTypesFailure, engineerrors.ErrActionFailed},
		{"skipped", db.RemediationStatusTypesSkipped, engineerrors.ErrActionSkipped},
		{"not available", db.RemediationStatusTypesNotAvailable, engineerrors.ErrActionNotAvailable},
		{"pending", db.RemediationStatusTypesPending, engineerrors.ErrActionPending},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := &db.ListRuleEvaluationsByProfileIdRow{RemStatus: tc.status}
			if tc.expected == nil {
				require.NoError(t, RemediationStatusAsError(row))
			} else {
				require.ErrorIs(t, RemediationStatusAsError(row), tc.expected)
			}
		})
	}
}

func TestErrorAsAlertStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected db.AlertStatusTypes
	}{
		{"nil error", nil, db.AlertStatusTypesOn},
		{"action turned off", engineerrors.ErrActionTurnedOff, db.AlertStatusTypesOff},
		{"action failed", engineerrors.ErrActionFailed, db.AlertStatusTypesError},
		{"action skipped", engineerrors.ErrActionSkipped, db.AlertStatusTypesSkipped},
		{"action not available", engineerrors.ErrActionNotAvailable, db.AlertStatusTypesNotAvailable},
		{"other error", errors.New("other error"), db.AlertStatusTypesError},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, ErrorAsAlertStatus(tc.err))
		})
	}
}

func TestAlertStatusAsError(t *testing.T) {
	t.Parallel()

	require.ErrorContains(t, AlertStatusAsError(nil), "no previous alert state")

	tests := []struct {
		name     string
		status   db.AlertStatusTypes
		expected error
	}{
		{"on", db.AlertStatusTypesOn, nil},
		{"off", db.AlertStatusTypesOff, engineerrors.ErrActionTurnedOff},
		{"error", db.AlertStatusTypesError, engineerrors.ErrActionFailed},
		{"skipped", db.AlertStatusTypesSkipped, engineerrors.ErrActionSkipped},
		{"not available", db.AlertStatusTypesNotAvailable, engineerrors.ErrActionNotAvailable},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := &db.ListRuleEvaluationsByProfileIdRow{AlertStatus: tc.status}
			if tc.expected == nil {
				require.NoError(t, AlertStatusAsError(row))
			} else {
				require.ErrorIs(t, AlertStatusAsError(row), tc.expected)
			}
		})
	}
}
