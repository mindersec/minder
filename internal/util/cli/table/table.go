// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"

	"github.com/mindersec/minder/internal/util/cli/table/layouts"
)

const (
	// Simple defines the standard table layout used across the Minder CLI.
	Simple               = "simple"
	defaultTerminalWidth = 100
)

// Table is the interface for rendering CLI tables using the go-pretty engine.
type Table interface {
	AddRow(row ...string)
	AddRowWithColor(row ...layouts.ColoredColumn)
	Render()
	SeparateRows() Table
	SetAutoMerge(merge bool) Table
	SetEqualColumns(equal bool) Table
}

// New creates a new table.
func New(_ string, layout layouts.TableLayout, header []string) Table {
	t := table.NewWriter()

	switch layout {
	case layouts.Condensed:
		t.SetStyle(table.StyleLight) // Thin lines for tight spaces
	case layouts.Heavy:
		t.SetStyle(table.StyleBold) // Thick lines for importance
	case layouts.Default:
		t.SetStyle(table.StyleRounded)
	default:
		t.SetStyle(table.StyleRounded) // The standard Minder look
	}

	t.SetOutputMirror(os.Stdout)

	headerRow := make(table.Row, len(header))
	maxColWidths := make([]int, len(header))

	for i, h := range header {
		headerRow[i] = h
		maxColWidths[i] = utf8.RuneCountInString(h)
	}
	t.AppendHeader(headerRow)

	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = true
	t.Style().Options.SeparateHeader = true
	t.Style().Options.SeparateRows = true
	t.Style().Box.PaddingLeft = " "
	t.Style().Box.PaddingRight = " "

	return &goPrettyTable{
		t:            t,
		numCols:      len(header),
		headers:      header,
		maxColWidths: maxColWidths,
	}
}

type goPrettyTable struct {
	t            table.Writer
	autoMerge    bool
	equalColumns bool
	numCols      int
	headers      []string
	maxColWidths []int
}

func (l *goPrettyTable) SetAutoMerge(merge bool) Table {
	l.autoMerge = merge
	return l
}

func (l *goPrettyTable) SetEqualColumns(equal bool) Table {
	l.equalColumns = equal
	return l
}

func (l *goPrettyTable) updateWidths(row []string) {
	for i, val := range row {
		if i >= l.numCols {
			break
		}
		w := 0
		for _, line := range strings.Split(val, "\n") {
			lw := utf8.RuneCountInString(line)
			if lw > w {
				w = lw
			}
		}
		if w > l.maxColWidths[i] {
			l.maxColWidths[i] = w
		}
	}
}

func (l *goPrettyTable) AddRow(row ...string) {
	r := make(table.Row, len(row))
	for i, val := range row {
		r[i] = val
	}
	l.updateWidths(row)
	l.t.AppendRow(r)
}

func (l *goPrettyTable) AddRowWithColor(row ...layouts.ColoredColumn) {
	r := make(table.Row, len(row))
	rawRow := make([]string, len(row))

	for i, cell := range row {
		val := cell.Column
		rawRow[i] = val

		switch cell.Color {
		case layouts.ColorRed:
			val = text.FgRed.Sprint(val)
		case layouts.ColorGreen:
			val = text.FgGreen.Sprint(val)
		case layouts.ColorYellow:
			val = text.FgYellow.Sprint(val)
		}
		r[i] = val
	}

	l.updateWidths(rawRow)
	l.t.AppendRow(r)
}

func (l *goPrettyTable) SeparateRows() Table {
	l.t.Style().Options.SeparateRows = true
	return l
}

func (l *goPrettyTable) Render() {
	w := getTerminalWidth()

	// With DrawBorder = false and SeparateColumns = true:
	barsAndPadding := (l.numCols - 1) + (l.numCols * 2)
	usableWidth := w - barsAndPadding
	if usableWidth < 10 {
		usableWidth = 10
	}

	assignedWidths := make([]int, l.numCols)

	if l.equalColumns && l.numCols > 0 {
		equalWidth := usableWidth / l.numCols
		for i := range assignedWidths {
			assignedWidths[i] = equalWidth
		}
		assignedWidths[l.numCols-1] += (usableWidth % l.numCols)
	} else {
		totalRequested := 0
		for _, req := range l.maxColWidths {
			totalRequested += req
		}

		if totalRequested > 0 {
			currentTotal := 0
			for i, req := range l.maxColWidths {
				// Calculate share: (column_req / total_req) * usable_width
				share := int(float64(req) / float64(totalRequested) * float64(usableWidth))

				// Ensure a minimum width so columns don't disappear
				if share < 5 {
					share = 5
				}
				assignedWidths[i] = share
				currentTotal += share
			}

			// Adjust for rounding errors to ensure we hit EXACTLY the terminal width
			diff := usableWidth - currentTotal
			if l.numCols > 0 {
				// If we have a deficit or surplus due to integer rounding,
				// adjust the last column to snap to the edge.
				if assignedWidths[l.numCols-1]+diff > 0 {
					assignedWidths[l.numCols-1] += diff
				}
			}
		} else {
			// Fallback if no rows were added: default to equal
			equalWidth := usableWidth / l.numCols
			for i := range assignedWidths {
				assignedWidths[i] = equalWidth
			}
		}
	}

	configs := make([]table.ColumnConfig, len(l.headers))
	for i := range l.headers {
		configs[i] = table.ColumnConfig{
			Number:           i + 1,
			WidthMax:         assignedWidths[i],
			WidthMin:         assignedWidths[i], // Forcing Min to match Max ensures full stretch
			WidthMaxEnforcer: text.WrapSoft,
			AutoMerge:        l.autoMerge,
			VAlign:           text.VAlignTop,
		}
	}

	l.t.SetColumnConfigs(configs)

	l.t.SetAllowedRowLength(w)

	l.t.Render()
}

func getTerminalWidth() int {
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if w, err := strconv.Atoi(cols); err == nil && w > 0 {
			return w
		}
	}

	//nolint:gosec // G115
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return defaultTerminalWidth
}
