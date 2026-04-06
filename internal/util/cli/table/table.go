// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	// Simple defines the standard table layout used across the Minder CLI.
	Simple               = "simple"
	defaultTerminalWidth = 80
)

// ColumnSpec defines how a specific column should be rendered universally.
type ColumnSpec struct {
	FixedWidth int
	WidthPct   float64
	AutoMerge  bool
}

var columnSpecs = map[string]ColumnSpec{
	// High-level grouping columns (AutoMerge: true)
	"Owner":       {WidthPct: 0.20, AutoMerge: true},
	"Provider":    {WidthPct: 0.25, AutoMerge: true},
	"Entity":      {WidthPct: 0.25, AutoMerge: true},
	"Time":        {FixedWidth: 20, AutoMerge: true},
	"Entity Type": {FixedWidth: 15, AutoMerge: true},
	"Rule":        {WidthPct: 0.25, AutoMerge: true},
	"Status":      {FixedWidth: 12, AutoMerge: true},

	// Identifying / Data Columns (AutoMerge: false)
	"Name":            {WidthPct: 0.35, AutoMerge: false},
	"Upstream ID":     {FixedWidth: 15, AutoMerge: false},
	"Rule Name":       {WidthPct: 0.25, AutoMerge: false},
	"Description":     {WidthPct: 0.60, AutoMerge: false},
	"Rule Params":     {WidthPct: 0.25, AutoMerge: false},
	"Rule Definition": {WidthPct: 0.50, AutoMerge: false},
	"Alert":           {FixedWidth: 10, AutoMerge: false},
	"Remediate":       {FixedWidth: 10, AutoMerge: false},
	"Details":         {WidthPct: 0.50, AutoMerge: false},
	"Evaluated At":    {FixedWidth: 25, AutoMerge: false},
}

// Table is an interface for rendering tables
type Table interface {
	AddRow(row ...string)
	AddRowWithColor(row ...layouts.ColoredColumn)
	Render()
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

	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = false
	t.Style().Format.Header = text.FormatDefault

	t.Style().Box.PaddingLeft = " "
	t.Style().Box.PaddingRight = " "

	return &goPrettyTable{
		t:       t,
		numCols: len(header),
		headers: header,
	}
}

type goPrettyTable struct {
	t         table.Writer
	autoMerge bool
	numCols   int
	headers   []string
}

// SetAutoMerge dynamically distributes column widths and assigns merge behavior
// based on the columnSpecs configuration.
func (l *goPrettyTable) SetAutoMerge(merge bool) {
	l.autoMerge = merge
	w := getTerminalWidth()
	usableWidth := w - 15

	fixedWidths := 0
	percentTotal := 0.0

	// Tally up fixed widths and percent weights based on headers
	for _, h := range l.headers {
		if spec, exists := columnSpecs[h]; exists {
			if spec.FixedWidth > 0 {
				fixedWidths += spec.FixedWidth
			} else if spec.WidthPct > 0 {
				percentTotal += spec.WidthPct
			}
		} else {
			// Unknown columns get a generic default weight
			percentTotal += 1.0
		}
	}

	remainingWidth := usableWidth - fixedWidths
	if remainingWidth < 10 {
		remainingWidth = 10 // Prevent negative widths on extremely small terminals
	}

	var configs []table.ColumnConfig
	// Generate the configuration for each column dynamically
	for i, h := range l.headers {
		cfg := table.ColumnConfig{
			Number:           i + 1,
			WidthMaxEnforcer: text.WrapText,
		}

		spec, known := columnSpecs[h]
		if known {
			// Assign width
			if spec.FixedWidth > 0 {
				cfg.WidthMax = spec.FixedWidth
			} else {
				weight := spec.WidthPct
				if percentTotal > 0 {
					weight = weight / percentTotal // Normalize percentages
				}
				cfg.WidthMax = int(float64(remainingWidth) * weight)
			}
			// Assign merge capability
			cfg.AutoMerge = merge && spec.AutoMerge
		} else {
			// Fallback for completely unknown columns
			weight := 1.0 / percentTotal
			cfg.WidthMax = int(float64(remainingWidth) * weight)
			cfg.AutoMerge = false
		}

		if cfg.AutoMerge {
			cfg.VAlign = text.VAlignTop
		}

		configs = append(configs, cfg)
	}
	l.t.SetColumnConfigs(configs)
}

func getTerminalWidth() int {
	fdPtr := os.Stdout.Fd()
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

// SeparateRows ensures each row is clearly separated
func (l *goPrettyTable) SeparateRows() {
	l.t.Style().Options.SeparateRows = true
}

func (l *goPrettyTable) Render() {
	w := getTerminalWidth()
	l.t.SetAllowedRowLength(w - 5)
	l.t.Render()
}
