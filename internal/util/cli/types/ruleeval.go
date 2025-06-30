// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/mindersec/minder/internal/util/cli/table"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type ruleEvalToDisplay struct {
	ruleEval *minderv1.RuleEvaluationStatus
}

// GetStatus implements table.EvalStatus.
func (r *ruleEvalToDisplay) GetStatus() string {
	return r.ruleEval.GetStatus()
}

// GetStatusDetail implements table.EvalStatus.
func (r *ruleEvalToDisplay) GetStatusDetail() string {
	return r.ruleEval.GetDetails()
}

// GetRemediationStatus implements table.EvalStatus.
func (r *ruleEvalToDisplay) GetRemediationStatus() string {
	return r.ruleEval.GetRemediationStatus()
}

// GetRemediationDetail implements table.EvalStatus.
func (r *ruleEvalToDisplay) GetRemediationDetail() string {
	return r.ruleEval.GetRemediationDetails()
}

// GetAlert implements table.EvalStatus.
func (r *ruleEvalToDisplay) GetAlert() table.StatusDetails {
	return r.ruleEval.GetAlert()
}

var _ table.EvalStatus = (*ruleEvalToDisplay)(nil)

// RuleEvalStatus converts a RuleEvaluationStatus for status display.
func RuleEvalStatus(ruleEval *minderv1.RuleEvaluationStatus) table.EvalStatus {
	return &ruleEvalToDisplay{ruleEval: ruleEval}
}
