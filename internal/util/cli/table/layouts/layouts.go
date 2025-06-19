// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// TODO: retire this entirely in favor of more native lipgloss

// Package layouts defines the available table layouts
package layouts

// TableLayout is the type for table layouts
type TableLayout string

const (
	// Default is the default table layout
	Default TableLayout = ""
)

// Color is the type for table colors
type Color string

const (
	// ColorRed is the color red
	ColorRed Color = "red"
	// ColorGreen is the color green
	ColorGreen Color = "green"
	// ColorYellow is the color yellow
	ColorYellow Color = "yellow"
)

// ColoredColumn is a column with a color
type ColoredColumn struct {
	Column string
	Color  Color
}

// RowsFromColoredColumns returns the rows of the colored columns
func RowsFromColoredColumns(c []ColoredColumn) []string {
	var columns []string
	for _, col := range c {
		columns = append(columns, col.Column)
	}
	return columns
}

// RedColumn returns a red colored column
func RedColumn(column string) ColoredColumn {
	return ColoredColumn{
		Column: column,
		Color:  ColorRed,
	}
}

// GreenColumn returns a green colored column
func GreenColumn(column string) ColoredColumn {
	return ColoredColumn{
		Column: column,
		Color:  ColorGreen,
	}
}

// YellowColumn returns a yellow colored column
func YellowColumn(column string) ColoredColumn {
	return ColoredColumn{
		Column: column,
		Color:  ColorYellow,
	}
}

// NoColor returns a column with no color
func NoColor(column string) ColoredColumn {
	return ColoredColumn{
		Column: column,
		Color:  "",
	}
}
