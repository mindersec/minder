// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package engine defines domain-level types used by the engine layer,
// decoupled from database representations.
package engine

// EvalStatus represents the result of a rule evaluation in the engine layer.
type EvalStatus string

const (
	// EvalStatusSuccess indicates the evaluation succeeded.
	EvalStatusSuccess EvalStatus = "success"

	// EvalStatusFailure indicates the evaluation failed.
	EvalStatusFailure EvalStatus = "failure"

	// EvalStatusError indicates an error occurred during evaluation.
	EvalStatusError EvalStatus = "error"

	// EvalStatusSkipped indicates the evaluation was skipped.
	EvalStatusSkipped EvalStatus = "skipped"
)

// EvaluationSnapshot represents a database-independent snapshot of evaluation state.
type EvaluationSnapshot struct {
	EvalStatus        EvalStatus
	RemediationStatus string
	AlertStatus       string
}
