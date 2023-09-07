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

package policy_status

import (
	"fmt"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const (
	successStatus = "success"
	failureStatus = "failure"
	errorStatus   = "error"
	skippedStatus = "skipped"
	pendingStatus = "pending"
)

func initializePolicyStatusTable(cmd *cobra.Command) *tablewriter.Table {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{"Id", "Name", "Overall Status", "Last Updated"})
	table.SetRowLine(true)
	table.SetRowSeparator("-")
	table.SetAutoWrapText(true)
	table.SetReflowDuringAutoWrap(true)

	return table
}

func renderPolicyStatusTable(
	ps *pb.PolicyStatus,
	table *tablewriter.Table,
) {
	row := []string{
		fmt.Sprintf("%d", ps.PolicyId),
		ps.PolicyName,
		getStatusText(ps.PolicyStatus),
		ps.LastUpdated.AsTime().Format(time.RFC3339),
	}

	table.Rich(row, []tablewriter.Colors{
		{},
		{},
		getStatusColor(ps.PolicyStatus),
		{},
	})
}

func initializeRuleEvaluationStatusTable(cmd *cobra.Command) *tablewriter.Table {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{
		"Policy ID", "Rule ID", "Rule Name", "Entity", "Status", "Entity Info", "Guidance"})
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
		fmt.Sprintf("%d", reval.PolicyId),
		fmt.Sprintf("%d", reval.RuleId),
		reval.RuleName,
		reval.Entity,
		getStatusText(reval.Status),
		mapToYAMLOrEmpty(reval.EntityInfo),
		guidanceOrEncouragement(reval.Status, reval.Guidance),
	}

	table.Rich(row, []tablewriter.Colors{
		{},
		{},
		{},
		{},
		getStatusColor(reval.Status),
		{},
		{},
	})
}

// Gets a friendly status text with an emoji
func getStatusText(status string) string {
	// statuses can be 'success', 'failure', 'error', 'skipped', 'pending'
	switch strings.ToLower(status) {
	case successStatus:
		return "✅ Success"
	case failureStatus:
		return "❌ Failure"
	case errorStatus:
		return "❌ Error"
	case skippedStatus:
		return "⚠️ Skipped"
	case pendingStatus:
		return "⏳ Pending"
	default:
		return "⚠️ Unknown"
	}
}

func getStatusColor(status string) tablewriter.Colors {
	// statuses can be 'success', 'failure', 'error', 'skipped', 'pending'
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

	return guidance
}
