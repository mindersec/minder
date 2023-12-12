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
	header     []table.Column
	rows       []table.Row
	valueWidth int
	tableWidth int
}

// New creates a new table with the given header
func New(layout string, header []string) *Table {
	var columns []table.Column
	tableWidth, columnWidth := getTableAndColumnWidths(layout, header)

	switch layout {
	case "keyvalue":
		columns = keyValueLayout(columnWidth)
	case "repolist":
		columns = repoListLayout()
	default:
		columns = defaultLayout(header, columnWidth)
	}
	return &Table{
		header:     columns,
		tableWidth: tableWidth,
		valueWidth: columnWidth,
	}
}

func getTableAndColumnWidths(layout string, header []string) (int, int) {
	tableWidth := cli.DefaultBannerWidth
	columnWidth := tableWidth - KeyWidth - 6
	if header != nil && layout == "" {
		columnWidth = (tableWidth / len(header)) - 3
	}
	return tableWidth, columnWidth
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
	r := table.New(
		table.WithColumns(t.header),
		table.WithRows(t.rows),
		table.WithFocused(false),
		table.WithHeight(len(t.rows)),
		table.WithStyles(TableHiddenSelectStyles),
	)
	fmt.Println(MainTableStyle.Render(r.View()))
}

func keyValueLayout(valueWidth int) []table.Column {
	return []table.Column{
		{Title: "Key", Width: KeyWidth},
		{Title: "Value", Width: valueWidth},
	}
}

func repoListLayout() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 40},
		{Title: "Project", Width: 40},
		{Title: "Provider", Width: 15},
		{Title: "Upstream ID", Width: 15},
		{Title: "Owner", Width: 15},
		{Title: "Name", Width: 15},
	}
}

func defaultLayout(header []string, valueWidth int) []table.Column {
	var columns []table.Column
	for i := range header {
		columns = append(columns, table.Column{
			Title: header[i],
			Width: valueWidth,
		})
	}
	return columns
}
