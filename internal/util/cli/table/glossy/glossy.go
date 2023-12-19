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

// Package glossy contains a glossy table
package glossy

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/stacklok/minder/internal/util/cli"
)

// KeyWidth is the width of the key column in the key value layout
const KeyWidth = 15

var (
	// MainTableStyle is the style to use for tables
	MainTableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(cli.AccentColor)
	// TableStyles is the style to use for tables
	TableStyles = table.Styles{
		Selected: lipgloss.NewStyle().Bold(true).Foreground(cli.SecondaryColor),
		Header:   lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(cli.PrimaryColor),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
	}
	// TableHiddenSelectStyles is the style to use for tables. It hides the selection
	// indicator.
	TableHiddenSelectStyles = table.Styles{
		Header:   lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(cli.PrimaryColor),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
		Selected: lipgloss.NewStyle(),
	}
)

// Table is a wrapper around tablewriter.Table
type Table struct {
	header      []table.Column
	rows        []table.Row
	columnWidth int
	tableWidth  int
}

// New creates a new table with the given header
func New(layout string, header []string) *Table {
	var columns []table.Column
	var columnWidth int

	tableWidth := cli.DefaultBannerWidth
	switch layout {
	case "keyvalue":
		columns, columnWidth = defaultLayout([]string{"Key", "Value"}, tableWidth)
	case "repolist":
		columns, columnWidth = defaultLayout([]string{"ID", "Project", "Provider", "Upstream ID", "Owner", "Name"}, tableWidth)
	case "ruletype":
		columns, columnWidth = defaultLayout([]string{"Provider", "Project", "ID", "Name", "Description"}, tableWidth)
	case "profile":
		columns, columnWidth = defaultLayout([]string{
			"Id", "Name", "Provider", "Entity", "Rule", "Rule Params", "Rule Definition"}, tableWidth)
	case "profile_status":
		columns, columnWidth = defaultLayout([]string{"Id", "Name", "Overall Status", "Last Updated"}, tableWidth)
	case "rule_evaluations":
		columns, columnWidth = defaultLayout([]string{
			"Rule ID", "Rule Name", "Entity", "Status", "Remediation Status", "Entity Info", "Guidance"}, tableWidth)
	default:
		columns, columnWidth = defaultLayout(header, tableWidth)
	}
	return &Table{
		header:      columns,
		tableWidth:  tableWidth,
		columnWidth: columnWidth,
	}
}

// AddRow adds a row
func (t *Table) AddRow(row []string) {
	t.rows = append(t.rows, row)
}

// AddRowWithColor adds a row with the given colors
func (t *Table) AddRowWithColor(row []string, _ []string) {
	t.rows = append(t.rows, row)
}

// Render renders the table
func (t *Table) Render() {
	// resize if needed
	for _, row := range t.rows {
		for i := range row {
			t.header[i].Width = t.columnWidth
		}
	}
	r := table.New(
		table.WithColumns(t.header),
		table.WithRows(t.rows),
		table.WithFocused(false),
		table.WithHeight(len(t.rows)),
		table.WithStyles(TableHiddenSelectStyles),
	)
	fmt.Println(MainTableStyle.Render(r.View()))
}

func defaultLayout(header []string, tableWidth int) ([]table.Column, int) {
	var columns []table.Column
	columnWidth := (tableWidth / len(header)) - 3
	for i := range header {
		columns = append(columns, table.Column{
			Title: header[i],
			Width: columnWidth,
		})
	}
	return columns, columnWidth
}
