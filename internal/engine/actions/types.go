// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package actions

import "encoding/json"

// RemediationStatus represents remediation state.
type RemediationStatus string

// RemediationStatus constants represent remediation states.
const (
	RemediationStatusSuccess      RemediationStatus = "success"
	RemediationStatusFailure      RemediationStatus = "failure"
	RemediationStatusError        RemediationStatus = "error"
	RemediationStatusSkipped      RemediationStatus = "skipped"
	RemediationStatusNotAvailable RemediationStatus = "not_available"
	RemediationStatusPending      RemediationStatus = "pending"
)

// AlertStatus represents alert state.
type AlertStatus string

// AlertStatus constants represent alert states.
const (
	AlertStatusOn           AlertStatus = "on"
	AlertStatusOff          AlertStatus = "off"
	AlertStatusError        AlertStatus = "error"
	AlertStatusSkipped      AlertStatus = "skipped"
	AlertStatusNotAvailable AlertStatus = "not_available"
)

// EvalStatus represents evaluation status.
type EvalStatus string

// EvalStatus constants represent evaluation statuses.
const (
	EvalStatusSuccess EvalStatus = "success"
	EvalStatusFailure EvalStatus = "failure"
	EvalStatusError   EvalStatus = "error"
	EvalStatusSkipped EvalStatus = "skipped"
	EvalStatusPending EvalStatus = "pending"
)

// previousEval captures previous remediation and alert state.
type previousEval struct {
	RemediationStatus RemediationStatus
	AlertStatus       AlertStatus
	RemediationMeta   json.RawMessage
	AlertMeta         json.RawMessage
}
