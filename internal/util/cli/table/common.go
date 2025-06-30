// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"strings"

	"github.com/mindersec/minder/internal/util/cli/table/layouts"
)

const (
	successStatus      = "success"
	failureStatus      = "failure"
	errorStatus        = "error"
	skippedStatus      = "skipped"
	pendingStatus      = "pending"
	notAvailableStatus = "not_available"
	onStatus           = "on"
	offStatus          = "off"
)

// StatusDetails provides an abstract interface for reading status details.
// This is currently only used for Alert status, because those structures
// consistently expose this pattern.
type StatusDetails interface {
	GetStatus() string
	GetDetails() string
}

// EvalStatus provides an abstract interface for reading evaluation status.
// Since we don't have common shapes between e.g. EvaluationHistory and
// RuleEvaluationStatus, we use an interface modeled on RuleEvaluationStatus
// to adapt the two.  (We would use the EvaluationHistory shape, but the
// sub-message return values are structs that don't match the interface here.)
type EvalStatus interface {
	GetStatus() string
	GetStatusDetail() string
	GetRemediationStatus() string
	GetRemediationDetail() string
	GetAlert() StatusDetails
}

type statusDisplay struct {
	Emoji    string
	Text     string
	Severity int // 0 = no color, 1 = green, 2 = yellow, 3 = red
}

var statuses = map[string]statusDisplay{
	"in compliance": {
		Emoji:    "✅",
		Text:     "Ok",
		Severity: 1,
	},
	"failed to evaluate": {
		Emoji:    "🚧",
		Text:     "Error",
		Severity: 3,
	},
	"skipped": {
		Emoji:    "➖",
		Text:     "Skipped",
		Severity: 2,
	},
	"fail no fix": {
		Emoji:    "⛔",
		Text:     "Failed",
		Severity: 3,
	},
	"remediated": {
		Emoji:    "🔧",
		Text:     "Fixed",
		Severity: 1,
	},
	"remediation failed": {
		Emoji:    "⛓️‍💥",
		Text:     "No fix",
		Severity: 3,
	},
	"remediation error": {
		Emoji:    "😷",
		Text:     "!Fix",
		Severity: 3,
	},
	"alert on": {
		Emoji:    "🚨",
		Text:     "Alert",
		Severity: 2,
	},
	"alert error": {
		Emoji:    "🤮",
		Text:     "!Alert",
		Severity: 3,
	},
	"unknown": {
		Emoji:    "❓",
		Text:     "Unknown",
		Severity: 0,
	},
}

// GetStatusIcon returns a colored column with the status icon for the given
// evaluation status.
//
//nolint:gocyclo
func GetStatusIcon(eval EvalStatus, emoji bool) layouts.ColoredColumn {
	results := []statusDisplay{}
	switch eval.GetStatus() {
	case successStatus:
		results = append(results, statuses["in compliance"])
	case errorStatus:
		results = append(results, statuses["failed to evaluate"])
	case skippedStatus:
		results = append(results, statuses["skipped"])
	case failureStatus:
		switch eval.GetRemediationStatus() {
		case successStatus:
			results = append(results, statuses["remediated"])
		case failureStatus:
			results = append(results, statuses["remediation failed"])
		case errorStatus:
			results = append(results, statuses["remediation error"])
		case skippedStatus, notAvailableStatus:
			// do nothing, we don't want to add an icon for these
		}
		switch eval.GetAlert().GetStatus() {
		case onStatus:
			results = append(results, statuses["alert on"])
		case errorStatus:
			results = append(results, statuses["alert error"])
		case offStatus, skippedStatus, notAvailableStatus:
			// do nothing, we don't want to add an icon for these
		}
		if len(results) == 0 {
			results = []statusDisplay{statuses["fail no fix"]}
		}
	default:
		results = []statusDisplay{statuses["unknown"]}
	}
	severity := 0
	tokens := []string{}
	separator := "" // Empty separator means emojis will be used
	if !emoji {
		separator = " "
	}
	for _, result := range results {
		if result.Severity > severity {
			severity = result.Severity
		}

		if emoji {
			tokens = append(tokens, result.Emoji)
		} else {
			tokens = append(tokens, result.Text)
		}
	}
	colorFunc := map[int]func(string) layouts.ColoredColumn{
		0: layouts.NoColor,
		1: layouts.GreenColumn,
		2: layouts.YellowColumn,
		3: layouts.RedColumn,
	}[severity]

	return colorFunc(strings.Join(tokens, separator))
}

// BestDetail returns the best detail for the given evaluation status.
func BestDetail(eval EvalStatus) layouts.ColoredColumn {
	// TODO: combine with GetStatusIcon, and pick color, etc based on status
	if eval.GetRemediationDetail() != "" {
		return layouts.NoColor(eval.GetRemediationDetail())
	}
	if eval.GetAlert().GetDetails() != "" {
		return layouts.NoColor(eval.GetAlert().GetDetails())
	}
	return layouts.NoColor(eval.GetStatusDetail())
}
