// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package actions contains core engine logic for processing rule actions
// such as remediation and alerts. This file defines engine-level types
// to decouple business logic from database representations.
package actions

import "encoding/json"

// EvalStatus represents the evaluation result status in the engine layer.
// This is intentionally decoupled from database-specific enums.
type EvalStatus string

const (
	// EvalStatusSuccess indicates a successful evaluation.
	EvalStatusSuccess EvalStatus = "success"

	// EvalStatusFailure indicates a failed evaluation.
	EvalStatusFailure EvalStatus = "failure"

	// EvalStatusError indicates an error occurred during evaluation.
	EvalStatusError EvalStatus = "error"

	// EvalStatusSkipped indicates the evaluation was skipped.
	EvalStatusSkipped EvalStatus = "skipped"

	// EvalStatusPending indicates the evaluation is pending.
	EvalStatusPending EvalStatus = "pending"
)

// RemediationStatus represents the remediation state in the engine layer.
// This type abstracts database-specific remediation status values.
type RemediationStatus string

// AlertStatus represents the alert state in the engine layer.
// This type abstracts database-specific alert status values.
type AlertStatus string

// PreviousEval represents the previous evaluation state used by the engine
// to determine action decisions. This structure replaces direct usage of
// database row types in the engine layer.
type PreviousEval struct {
	// RemediationStatus holds the previous remediation status.
	RemediationStatus RemediationStatus

	// AlertStatus holds the previous alert status.
	AlertStatus AlertStatus

	// RemediationMeta contains metadata related to remediation actions.
	RemediationMeta *json.RawMessage

	// AlertMeta contains metadata related to alert actions.
	AlertMeta *json.RawMessage
}
