// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestFormatEvaluationReasoning(t *testing.T) {
	t.Parallel()

	eval := &minderv1.RuleEvaluationStatus{
		Details:             "the base check failed",
		Guidance:            "rebuild with the required settings",
		RemediationDetails:  "apply the remediation workflow",
		RemediationUrl:      "https://example.com/remediate",
		RuleDescriptionName: "Require artifact attestation",
		RuleTypeName:        "artifact_attestation_slsa",
		Status:              "failure",
		Alert: &minderv1.EvalResultAlert{
			Status:  "on",
			Details: "alert is enabled",
			Url:     "https://example.com/alert",
		},
	}

	assert.Equal(t, "Alert: alert is enabled\nURL: https://example.com/alert\nRemediation: apply the remediation workflow\nURL: https://example.com/remediate\nDetails: the base check failed\nGuidance: rebuild with the required settings", FormatEvaluationReasoning(eval))
}

func TestFormatEvaluationReasoning_Empty(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "-", FormatEvaluationReasoning(&minderv1.RuleEvaluationStatus{}))
}
