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

// Package layouts defines the available table layouts
package layouts

import "github.com/olekukonko/tablewriter"

// TableLayout is the type for table layouts
type TableLayout string

const (
	// KeyValue is the key value table layout
	KeyValue TableLayout = "keyvalue"
	// RuleTypeOne is the rule type table layout
	RuleTypeOne TableLayout = "ruletype"
	// RuleTypeList is the rule type table layout
	RuleTypeList TableLayout = "ruletype_list"
	// ProfileSettings is the profile settings table layout
	ProfileSettings TableLayout = "profile_settings"
	// Profile is the profile table layout
	Profile TableLayout = "profile"
	// ProviderList is the provider list table layout
	ProviderList TableLayout = "provider_list"
	// RepoList is the repo list table layout
	RepoList TableLayout = "repolist"
	// ProfileStatus is the profile status table layout
	ProfileStatus TableLayout = "profile_status"
	// RuleEvaluations is the rule evaluations table layout
	RuleEvaluations TableLayout = "rule_evaluations"
	// EvaluationHistory is the evaluation history table layout
	EvaluationHistory TableLayout = "evaluation_history"
	// RoleList is the roles list table layout
	RoleList TableLayout = "role_list"
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

// ColorsFromColoredColumns returns the colors of the colored columns
func ColorsFromColoredColumns(r []ColoredColumn) []tablewriter.Colors {
	colors := make([]tablewriter.Colors, len(r))
	for i := range r {
		c := r[i].Color
		switch c {
		case ColorRed:
			colors[i] = tablewriter.Colors{tablewriter.FgRedColor}
		case ColorGreen:
			colors[i] = tablewriter.Colors{tablewriter.FgGreenColor}
		case ColorYellow:
			colors[i] = tablewriter.Colors{tablewriter.FgYellowColor}
		default:
			colors[i] = tablewriter.Colors{}
		}
	}

	return colors
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
