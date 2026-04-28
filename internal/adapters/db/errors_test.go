// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0
package dbadapter

import (
	"testing"

	"github.com/mindersec/minder/internal/db"
)

func TestEvalHelpers(t *testing.T) {
	t.Parallel()

	if !IsEvalFailure(db.EvalStatusTypesFailure) {
		t.Error("expected failure")
	}
	if !IsEvalSuccess(db.EvalStatusTypesSuccess) {
		t.Error("expected success")
	}
	if !IsEvalError(db.EvalStatusTypesError) {
		t.Error("expected error")
	}
}

func TestRemediationHelpers(t *testing.T) {
	t.Parallel()

	if !IsRemediationSkipped(db.RemediationStatusTypesSkipped) {
		t.Error("expected skipped")
	}
}

func TestAlertHelpers(t *testing.T) {
	t.Parallel()

	if !IsAlertOn(db.AlertStatusTypesOn) {
		t.Error("expected alert on")
	}
	if !IsAlertOff(db.AlertStatusTypesOff) {
		t.Error("expected alert off")
	}
}
