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

	"github.com/stacklok/minder/internal/util/cli/table/layouts"
)

// Table is a wrapper around tablewriter.Table
type Table struct {
	table *tablewriter.Table
}

// New creates a new table with the given header
func New(layout layouts.TableLayout, header []string) *Table {
	table := tablewriter.NewWriter(os.Stdout)
	switch layout {
	case layouts.KeyValue:
		keyValueLayout(table)
	case layouts.RuleTypeOne:
		ruleTypeLayout(table)
	case layouts.RuleTypeList:
		ruleTypeListLayout(table)
	case layouts.ProfileSettings:
		profileSettingsLayout(table)
	case layouts.Profile:
		profileLayout(table)
	case layouts.ProviderList:
		providerListLayout(table)
	case layouts.RepoList:
		repoListLayout(table)
	case layouts.ProfileStatus:
		profileStatusLayout(table)
	case layouts.RuleEvaluations:
		ruleEvaluationsLayout(table)
	case layouts.RoleList:
		roleListLayout(table)
	case layouts.EvaluationHistory:
		evaluationHistoryLayout(table)
	case layouts.Default:
		table.SetHeader(header)
		defaultLayout(table)
	default:
		table.SetHeader(header)
		defaultLayout(table)
	}
	return &Table{
		table: table,
	}
}

// AddRow adds a row
func (t *Table) AddRow(row ...string) {
	t.table.Append(row)
}

// AddRowWithColor adds a row with the given colors
func (t *Table) AddRowWithColor(row ...layouts.ColoredColumn) {
	t.table.Rich(layouts.RowsFromColoredColumns(row), layouts.ColorsFromColoredColumns(row))
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
	table.SetHeader([]string{"ID", "Name", "Alert", "Remediate"})
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
	table.SetHeader([]string{"ID", "Name", "Status", "Last Updated"})
	table.SetReflowDuringAutoWrap(true)
}

func ruleEvaluationsLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{
		"Rule Type ID", "Rule Name", "Entity", "Status", "Remediation", "Entity Info"})
	table.SetAutoMergeCellsByColumnIndex([]int{0})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(true)
}

func repoListLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"ID", "Project", "Provider", "Upstream ID", "Owner", "Name"})
}

func ruleTypeListLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"Project", "ID", "Name", "Description"})
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 2, 3})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(true)
}

func ruleTypeLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"Rule Type", "Details"})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(false)
}

func roleListLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"Name", "Description"})
	table.SetAutoWrapText(false)
}

func providerListLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{"Name", "Project", "Version", "Implements"})
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 2, 3})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(false)
}

func evaluationHistoryLayout(table *tablewriter.Table) {
	defaultLayout(table)
	table.SetHeader([]string{
		"Time", "Rule", "Entity", "Status", "Remediation Status", "Alert Status"})
	table.SetAutoMergeCellsByColumnIndex([]int{0})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(true)
}
