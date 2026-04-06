// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package table contains utilities for rendering tables
package table

import (
	"math"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"

	"github.com/mindersec/minder/internal/util/cli/table/layouts"
)

const (
	// Simple is a simple table
	Simple               = "simple"
	defaultTerminalWidth = 80
)

// Table is an interface for rendering tables
type Table interface {
	AddRow(row ...string)
	AddRowWithColor(row ...layouts.ColoredColumn)
	// Render outputs the table to stdout (TODO: make output configurable)
	Render()
	// SeparateRows ensures each row is clearly separated
	SeparateRows()
	SetAutoMerge(merge bool)
}

// New creates a new table using the go-pretty engine
func New(_ string, _ layouts.TableLayout, header []string) Table {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	headerRow := make(table.Row, len(header))
	for i, h := range header {
		headerRow[i] = h
	}
	t.AppendHeader(headerRow)

	// Rounded style is the Minder standard.
	// SeparateRows is disabled to allow AutoMerge to create clean visual blocks.
	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = false
	t.Style().Format.Header = text.FormatDefault

	return &goPrettyTable{
		t:       t,
		numCols: len(header),
	}
}

type goPrettyTable struct {
	t         table.Writer
	autoMerge bool
	numCols   int
}

func (l *goPrettyTable) SetAutoMerge(merge bool) {
	l.autoMerge = merge

	w := getTerminalWidth()
	usableWidth := w - 10

	var configs []table.ColumnConfig
	for i := 1; i <= l.numCols; i++ {
		cfg := table.ColumnConfig{
			Number:           i,
			WidthMaxEnforcer: text.WrapSoft,
		}

		cfg.WidthMax = l.getColumnWidth(i, usableWidth)

		if i == 1 || i == 2 {
			cfg.AutoMerge = merge
			cfg.VAlign = text.VAlignTop
		}
		configs = append(configs, cfg)
	}
	l.t.SetColumnConfigs(configs)
}

func (l *goPrettyTable) getColumnWidth(colIdx int, totalWidth int) int {
	switch l.numCols {
	case 4:
		return getFourColWidth(colIdx, totalWidth)
	case 3:
		return getThreeColWidth(colIdx, totalWidth)
	default:
		return totalWidth / l.numCols
	}
}

func getFourColWidth(colIdx int, totalWidth int) int {
	// We use a mix of fixed widths for small data and percentages for large data.
	switch colIdx {
	case 1:
		return 20
	case 4:
		return 12
	case 2:
		return int(float64(totalWidth-32) * 0.40)
	case 3:
		return int(float64(totalWidth-32) * 0.60)
	default:
		return totalWidth / 4
	}
}

func getThreeColWidth(colIdx int, totalWidth int) int {
	switch colIdx {
	case 1:
		return int(float64(totalWidth) * 0.30)
	case 2:
		return int(float64(totalWidth) * 0.20)
	case 3:
		return int(float64(totalWidth) * 0.50)
	default:
		return totalWidth / 3
	}
}

func getTerminalWidth() int {
	fdPtr := os.Stdout.Fd()
	// G115 fix: safety check for uintptr to int conversion
	if fdPtr <= math.MaxInt {
		if w, _, err := term.GetSize(int(fdPtr)); err == nil && w > 0 {
			return w
		}
	}
	return defaultTerminalWidth
}

func (l *goPrettyTable) AddRow(row ...string) {
	r := make(table.Row, len(row))
	for i, val := range row {
		r[i] = val
	}
	l.t.AppendRow(r)
	// Append a horizontal line between blocks if merging is active
	if l.autoMerge {
		l.t.AppendSeparator()
	}
}

func (l *goPrettyTable) AddRowWithColor(row ...layouts.ColoredColumn) {
	r := make(table.Row, len(row))
	for i, cell := range row {
		val := cell.Column
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
	l.t.AppendRow(r)
	if l.autoMerge {
		l.t.AppendSeparator()
	}
}

func (l *goPrettyTable) SeparateRows() {
	l.t.Style().Options.SeparateRows = true
}

func (l *goPrettyTable) Render() {
	w := getTerminalWidth()
	// Forces the right-hand boundary line to snap to the terminal edge
	l.t.SetAllowedRowLength(w)
	l.t.Render()
}
