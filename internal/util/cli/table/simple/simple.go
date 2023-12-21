// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package simple contains a simple table
package simple

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

// Table is a wrapper around tablewriter.Table
type Table struct {
	table *tablewriter.Table
}

// New creates a new table with the given header
func New(layout string, header []string) *Table {
	table := tablewriter.NewWriter(os.Stdout)
	switch layout {
	case "keyvalue":
		keyValueLayout(table)
	case "ruletype":
		ruleTypeLayout(table)
	case "profile_settings":
		profileSettingsLayout(table)
	case "profile":
		profileLayout(table)
	case "repolist":
		repoListLayout(table)
	case "profile_status":
		profileStatusLayout(table)
	case "rule_evaluations":
		ruleEvaluationsLayout(table)
	default:
		table.SetHeader(header)
		defaultLayout(table)
	}
	return &Table{
		table: table,
	}
}

// AddRow adds a row
func (t *Table) AddRow(row []string) {
	t.table.Append(row)
}

// AddRowWithColor adds a row with the given colors
func (t *Table) AddRowWithColor(row []string, rowColors []string) {
	colors := make([]tablewriter.Colors, len(rowColors))
	for i := range rowColors {
		switch rowColors[i] {
		case "red":
			colors[i] = tablewriter.Colors{tablewriter.FgRedColor}
		case "green":
			colors[i] = tablewriter.Colors{tablewriter.FgGreenColor}
		case "yellow":
			colors[i] = tablewriter.Colors{tablewriter.FgYellowColor}
		default:
			colors[i] = tablewriter.Colors{}
		}
	}
	t.table.Rich(row, colors)
}

// Render renders the table
func (t *Table) Render() {
	t.table.Render()
}

func defaultLayout(table *tablewriter.Table) {
	table.SetRowLine(true)
	table.SetRowSeparator("-")
	table.SetAutoWrapText(true)
}

func keyValueLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"Key", "Value"})
	table.SetColMinWidth(0, 50)
	table.SetColMinWidth(1, 50)
}

func profileSettingsLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"Profile Summary"})
	table.SetColMinWidth(0, 50)
	table.SetColMinWidth(1, 50)
}

func profileLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"Entity", "Rule", "Rule Params", "Rule Definition"})
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(false)
}

func profileStatusLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"ID", "Name", "Overall Status", "Last Updated"})
	table.SetReflowDuringAutoWrap(true)
}

func ruleEvaluationsLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{
		"Rule ID", "Rule Name", "Entity", "Status", "Remediation Status", "Entity Info", "Guidance"})
	table.SetAutoMergeCellsByColumnIndex([]int{0})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(false)
}

func repoListLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"ID", "Project", "Provider", "Upstream ID", "Owner", "Name"})
}

func ruleTypeLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"Provider", "Project Name", "ID", "Name", "Description"})
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 2, 3})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(false)

}
