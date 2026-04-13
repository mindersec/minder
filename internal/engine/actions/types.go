// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package actions

import "encoding/json"

// RemediationStatus represents a high-level remediation state, decoupled
// from database-specific types.
type RemediationStatus string

// RemediationStatus* constants mirror the remediation status lifecycle used
// for actions while remaining decoupled from database-specific types.
const (
	RemediationStatusSuccess      RemediationStatus = "success"
	RemediationStatusFailure      RemediationStatus = "failure"
	RemediationStatusError        RemediationStatus = "error"
	RemediationStatusSkipped      RemediationStatus = "skipped"
	RemediationStatusNotAvailable RemediationStatus = "not_available"
	RemediationStatusPending      RemediationStatus = "pending"
)

// AlertStatus represents a high-level alert state, decoupled from
// database-specific types.
type AlertStatus string

// AlertStatus* constants mirror the alert status lifecycle used for actions
// while remaining decoupled from database-specific types.
const (
	AlertStatusOn           AlertStatus = "on"
	AlertStatusOff          AlertStatus = "off"
	AlertStatusError        AlertStatus = "error"
	AlertStatusSkipped      AlertStatus = "skipped"
	AlertStatusNotAvailable AlertStatus = "not_available"
)

// EvalStatus represents the normalized evaluation status derived from
// rule evaluation errors.
type EvalStatus string

// EvalStatus* constants represent the normalized evaluation status derived
// from rule evaluation errors.
const (
	EvalStatusSuccess EvalStatus = "success"
	EvalStatusFailure EvalStatus = "failure"
	EvalStatusError   EvalStatus = "error"
	EvalStatusSkipped EvalStatus = "skipped"
	EvalStatusPending EvalStatus = "pending"
)

// PreviousEval captures the previous remediation and alert state along with
// associated metadata in a database-agnostic form.
type PreviousEval struct {
	RemediationStatus RemediationStatus
	AlertStatus       AlertStatus
	RemediationMeta   *json.RawMessage
	AlertMeta         *json.RawMessage
}
