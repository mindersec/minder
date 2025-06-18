// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package table contains utilities for rendering tables
package table

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	lg "github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"

	"github.com/mindersec/minder/internal/util/cli/table/layouts"
)

const (
	// Simple is a simple table
	Simple = "simple"
)

// Table is an interface for rendering tables
type Table interface {
	AddRow(row ...string)
	AddRowWithColor(row ...layouts.ColoredColumn)
	// Render outputs the table to stdout (TODO: make output configurable)
	Render()
	// SeparateRows ensures each row is clearly separated (probably because it is multi-line)
	SeparateRows()
}

// New creates a new table
func New(_ string, _ layouts.TableLayout, header []string) Table {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w == 0 {
		w = 80 // Default width if we can't determine terminal size
	}
	return &lipGlossTable{
		table: lg.New().
			//Border(lipgloss.HiddenBorder()).
			BorderTop(false).BorderBottom(false).
			BorderLeft(false).BorderRight(false).
			Headers(header...).
			Width(w).Wrap(true),
		colors: make(map[int]map[int]layouts.Color),
	}
}

type lipGlossTable struct {
	table  *lg.Table
	rows   int
	colors map[int]map[int]layouts.Color
}

// AddRow implements Table.
func (l *lipGlossTable) AddRow(row ...string) {
	l.table.Row(row...)
	l.rows++
}

// AddRowWithColor implements Table.
func (l *lipGlossTable) AddRowWithColor(row ...layouts.ColoredColumn) {
	cells := make([]string, 0, len(row))
	for i, cell := range row {
		cells = append(cells, cell.Column)
		if cell.Color != "" {
			row := l.colors[l.rows]
			if row == nil {
				row = make(map[int]layouts.Color)
				l.colors[l.rows] = row
			}
			l.colors[l.rows][i] = cell.Color
		}
	}
	l.AddRow(cells...)
}

// SeparateRows implements Table.
func (l *lipGlossTable) SeparateRows() {
	l.table.BorderRow(true).
		BorderLeft(true).BorderRight(true).
		BorderTop(true).BorderBottom(true)
}

// Render implements Table.
func (l *lipGlossTable) Render() {
	l.table.StyleFunc(func(row, col int) lipgloss.Style {
		style := lipgloss.NewStyle().
			Padding(0, 1)
		rowData := l.colors[row]
		if rowData == nil {
			return style
		}
		color := rowData[col]
		switch color {
		case layouts.ColorRed:
			return style.Foreground(lipgloss.Color("#ff0000"))
		case layouts.ColorGreen:
			return style.Foreground(lipgloss.Color("#00ff00"))
		case layouts.ColorYellow:
			return style.Foreground(lipgloss.Color("#ffff00"))
		default:
			return style
		}
	})

	fmt.Println(l.table.Render())
}
