// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package common contains logic shared between multiple subcommands
package common

import (
	"strings"

	"github.com/stacklok/minder/internal/util/cli/table/layouts"
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

// GetEvalStatusColor maps the alert status to coloured text
func GetEvalStatusColor(status string) layouts.ColoredColumn {
	txt := getStatusText(status)
	// eval statuses can be 'success', 'failure', 'error', 'skipped', 'pending'
	switch strings.ToLower(status) {
	case successStatus:
		return layouts.GreenColumn(txt)
	case failureStatus:
		return layouts.RedColumn(txt)
	case errorStatus:
		return layouts.RedColumn(txt)
	case skippedStatus:
		return layouts.YellowColumn(txt)
	default:
		return layouts.NoColor(txt)
	}
}

// GetRemediateStatusColor maps the alert status to coloured text
func GetRemediateStatusColor(status string) layouts.ColoredColumn {
	txt := getStatusText(status)
	// remediation statuses can be 'success', 'failure', 'error', 'skipped', 'not supported'
	switch strings.ToLower(status) {
	case successStatus:
		return layouts.GreenColumn(txt)
	case failureStatus:
		return layouts.RedColumn(txt)
	case errorStatus:
		return layouts.RedColumn(txt)
	case notAvailableStatus, skippedStatus:
		return layouts.YellowColumn(txt)
	default:
		return layouts.NoColor(txt)
	}
}

// GetAlertStatusColor maps the alert status to coloured text
func GetAlertStatusColor(status string) layouts.ColoredColumn {
	txt := getStatusText(status)
	// alert statuses can be 'on', 'off', 'error', 'skipped', 'not available'
	switch strings.ToLower(status) {
	case onStatus:
		return layouts.GreenColumn(txt)
	case offStatus:
		return layouts.YellowColumn(txt)
	case errorStatus:
		return layouts.RedColumn(txt)
	case notAvailableStatus, skippedStatus:
		return layouts.YellowColumn(txt)
	default:
		return layouts.NoColor(txt)
	}
}

func getStatusText(status string) string {
	// remediation statuses can be 'success', 'failure', 'error', 'skipped', 'pending' or 'not supported'
	switch strings.ToLower(status) {
	case onStatus:
		return "Success"
	case offStatus:
		return "Failure"
	case successStatus:
		return "Success"
	case failureStatus:
		return "Failure"
	case errorStatus:
		return "Error"
	case skippedStatus:
		return "Skipped" // visually empty as we didn't have to remediate
	case pendingStatus:
		return "Pending"
	case notAvailableStatus:
		return "Not Available"
	default:
		return "Unknown"
	}
}
