// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package types provides normalization types for CLI rendering.
package types

import (
	"github.com/mindersec/minder/internal/util/cli/table"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type historyToDisplay struct {
	history *minderv1.EvaluationHistory
}

var _ table.EvalStatus = (*historyToDisplay)(nil)

// GetStatusDetail implements table.EvalStatus.
func (h *historyToDisplay) GetStatusDetail() string {
	return h.history.GetStatus().GetDetails()
}

// GetStatus implements table.EvalStatus.
func (h *historyToDisplay) GetStatus() string {
	return h.history.GetStatus().GetStatus()
}

// GetRemediationStatus implements table.EvalStatus.
func (h *historyToDisplay) GetRemediationStatus() string {
	return h.history.GetRemediation().GetStatus()
}

// GetRemediationDetail implements table.EvalStatus.
func (h *historyToDisplay) GetRemediationDetail() string {
	return h.history.GetRemediation().GetDetails()
}

// GetAlert implements table.EvalStatus.
func (h *historyToDisplay) GetAlert() table.StatusDetails {
	return h.history.GetAlert()
}

// HistoryStatus converts an EvaluationHistory for status display.
func HistoryStatus(history *minderv1.EvaluationHistory) table.EvalStatus {
	return &historyToDisplay{history: history}
}
