//
// Copyright 2023 Stacklok, Inc.
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

package profile_status

import (
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

const (
	successStatus      = "success"
	failureStatus      = "failure"
	errorStatus        = "error"
	skippedStatus      = "skipped"
	pendingStatus      = "pending"
	notAvailableStatus = "not_available"
)

func initializeProfileStatusTable(cmd *cobra.Command) *tablewriter.Table {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{"Id", "Name", "Overall Status", "Last Updated"})
	table.SetRowLine(true)
	table.SetRowSeparator("-")
	table.SetAutoWrapText(true)
	table.SetReflowDuringAutoWrap(true)

	return table
}

func renderProfileStatusTable(
	ps *pb.ProfileStatus,
	table *tablewriter.Table,
) {
	row := []string{
		ps.ProfileId,
		ps.ProfileName,
		getEvalStatusText(ps.ProfileStatus),
		ps.LastUpdated.AsTime().Format(time.RFC3339),
	}

	table.Rich(row, []tablewriter.Colors{
		{},
		{},
		getEvalStatusColor(ps.ProfileStatus),
		{},
	})
}

func initializeRuleEvaluationStatusTable(cmd *cobra.Command) *tablewriter.Table {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{
		"Rule ID", "Rule Name", "Entity", "Status", "Remediation Status", "Entity Info", "Guidance"})
	table.SetRowLine(true)
	table.SetRowSeparator("-")
	table.SetAutoMergeCellsByColumnIndex([]int{0})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(false)

	return table
}

func renderRuleEvaluationStatusTable(
	reval *pb.RuleEvaluationStatus,
	table *tablewriter.Table,
) {
	row := []string{
		reval.RuleId,
		reval.RuleName,
		reval.Entity,
		getEvalStatusText(reval.Status),
		getRemediationStatusText(reval.RemediationStatus),
		mapToYAMLOrEmpty(reval.EntityInfo),
		guidanceOrEncouragement(reval.Status, reval.Guidance),
	}

	table.Rich(row, []tablewriter.Colors{
		{},
		{},
		{},
		{},
		getEvalStatusColor(reval.Status),
		getRemediateStatusColor(reval.RemediationStatus),
		{},
		{},
	})
}

// Gets a friendly status text with an emoji
func getEvalStatusText(status string) string {
	// eval statuses can be 'success', 'failure', 'error', 'skipped', 'pending'
	switch strings.ToLower(status) {
	case successStatus:
		return "✅ Success"
	case failureStatus:
		return "❌ Failure"
	case errorStatus:
		return "❌ Error"
	case skippedStatus:
		return "⏹ Skipped"
	case pendingStatus:
		return "⏳ Pending"
	default:
		return "⚠️ Unknown"
	}
}

// Gets a friendly status text with an emoji
func getRemediationStatusText(status string) string {
	// remediation statuses can be 'success', 'failure', 'error', 'skipped', 'not supported'
	switch strings.ToLower(status) {
	case successStatus:
		return "✅ Success"
	case failureStatus:
		return "❌ Failure"
	case errorStatus:
		return "❌ Error"
	case skippedStatus:
		return "" // visually empty as we didn't have to remediate
	case notAvailableStatus:
		return "🚫 Not Available"
	default:
		return "⚠️ Unknown"
	}
}

func getEvalStatusColor(status string) tablewriter.Colors {
	// eval statuses can be 'success', 'failure', 'error', 'skipped', 'pending'
	switch strings.ToLower(status) {
	case successStatus:
		return tablewriter.Colors{tablewriter.FgGreenColor}
	case failureStatus:
		return tablewriter.Colors{tablewriter.FgRedColor}
	case errorStatus:
		return tablewriter.Colors{tablewriter.FgRedColor}
	case skippedStatus:
		return tablewriter.Colors{tablewriter.FgYellowColor}
	default:
		return tablewriter.Colors{}
	}
}

func getRemediateStatusColor(status string) tablewriter.Colors {
	// remediation statuses can be 'success', 'failure', 'error', 'skipped', 'not supported'
	switch strings.ToLower(status) {
	case successStatus:
		return tablewriter.Colors{tablewriter.FgGreenColor}
	case failureStatus:
		return tablewriter.Colors{tablewriter.FgRedColor}
	case errorStatus:
		return tablewriter.Colors{tablewriter.FgRedColor}
	case notAvailableStatus:
		return tablewriter.Colors{tablewriter.FgYellowColor}
	default:
		return tablewriter.Colors{}
	}
}

func mapToYAMLOrEmpty(m map[string]string) string {
	if m == nil {
		return ""
	}

	yamlText, err := yaml.Marshal(m)
	if err != nil {
		return ""
	}

	return string(yamlText)
}

func guidanceOrEncouragement(status, guidance string) string {
	if status == successStatus && guidance == "" {
		return "👍"
	}

	if guidance == "" {
		return "No guidance available for this rule 😞"
	}

	// TODO: use a color scheme for minder instead of a pre-defined one.
	// Related-to: https://github.com/stacklok/mediator/issues/1006
	renderedGuidance, err := glamour.Render(guidance, "dark")
	if err != nil {
		return guidance
	}

	return renderedGuidance
}
