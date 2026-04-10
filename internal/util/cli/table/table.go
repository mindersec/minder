// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"io"
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
	// SeparateRows ensures each row is clearly separated (probably because it is multi-line)
	SeparateRows() Table
	SetAutoMerge(merge bool) Table
	SetEqualColumns(equal bool) Table
}

// New creates a new table.
func New(_ string, layout layouts.TableLayout, out io.Writer, header []string) Table {
	t := table.NewWriter()

	switch layout {
	case layouts.Condensed:
		t.SetStyle(table.StyleLight) // Thin lines for tight spaces
	case layouts.Heavy:
		t.SetStyle(table.StyleBold) // Thick lines for importance
	case layouts.Default:
		t.SetStyle(table.StyleRounded) // Rounded corner for table
	default:
		t.SetStyle(table.StyleRounded) // The standard Minder look
	}

	t.SetOutputMirror(out)

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

func (g *goPrettyTable) SetAutoMerge(merge bool) Table {
	g.autoMerge = merge
	return g
}

func (g *goPrettyTable) SetEqualColumns(equal bool) Table {
	g.equalColumns = equal
	return g
}

func (g *goPrettyTable) updateWidths(row []string) {
	for i, val := range row {
		if i >= g.numCols {
			break
		}
		w := 0
		for _, line := range strings.Split(val, "\n") {
			lw := utf8.RuneCountInString(line)
			if lw > w {
				w = lw
			}
		}
		if w > g.maxColWidths[i] {
			g.maxColWidths[i] = w
		}
	}
}

func (g *goPrettyTable) AddRow(row ...string) {
	r := make(table.Row, len(row))
	for i, val := range row {
		r[i] = val
	}
	g.updateWidths(row)
	g.t.AppendRow(r)
}

func (g *goPrettyTable) AddRowWithColor(row ...layouts.ColoredColumn) {
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

	g.updateWidths(rawRow)
	g.t.AppendRow(r)
}

func (g *goPrettyTable) SeparateRows() Table {
	g.t.Style().Options.SeparateRows = true
	return g
}

func (g *goPrettyTable) Render() {
	w := getTerminalWidth()

	// With DrawBorder = false and SeparateColumns = true:
	barsAndPadding := (g.numCols - 1) + (g.numCols * 2)
	usableWidth := w - barsAndPadding
	if usableWidth < 10 {
		usableWidth = 10
	}

	assignedWidths := make([]int, g.numCols)

	if g.equalColumns && g.numCols > 0 {
		equalWidth := usableWidth / g.numCols
		for i := range assignedWidths {
			assignedWidths[i] = equalWidth
		}
		assignedWidths[g.numCols-1] += (usableWidth % g.numCols)
	} else {
		totalRequested := 0
		for _, req := range g.maxColWidths {
			totalRequested += req
		}

		if totalRequested > 0 {
			currentTotal := 0
			for i, req := range g.maxColWidths {
				// Calculate share: (column_req / total_req) * usable_width
				share := int(float64(req) / float64(totalRequested) * float64(usableWidth))

				// Ensure a minimum width so columns don't disappear
				if share < 5 {
					share = 5
				}
				assignedWidths[i] = share
				currentTotal += share
			}

			diff := usableWidth - currentTotal
			if g.numCols > 0 {
				if assignedWidths[g.numCols-1]+diff > 5 {
					assignedWidths[g.numCols-1] += diff
				} else {
					assignedWidths[g.numCols-1] = 5
				}
			}
		} else {
			// Fallback if no rows were added: default to equal
			equalWidth := usableWidth / g.numCols
			for i := range assignedWidths {
				assignedWidths[i] = equalWidth
			}
		}
	}

	configs := make([]table.ColumnConfig, len(g.headers))
	for i := range g.headers {
		configs[i] = table.ColumnConfig{
			Number:           i + 1,
			WidthMax:         assignedWidths[i],
			WidthMin:         assignedWidths[i], // Forcing Min to match Max ensures full stretch
			WidthMaxEnforcer: text.WrapSoft,
			AutoMerge:        g.autoMerge,
			VAlign:           text.VAlignTop,
		}
	}

	g.t.SetColumnConfigs(configs)

	g.t.Style().Size.WidthMax = w

	g.t.Render()
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
