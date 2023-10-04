//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package cli

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Color Palette
var (
	// PrimaryColor is the primary color for the cli.
	PrimaryColor = lipgloss.Color("#00BBBE")
	// Secondary is the secondary color for the cli.
	SecondaryColor = lipgloss.Color("#59CFA8")
	// AccentColor is the accent color for the cli.
	AccentColor = lipgloss.Color("#3D34E0")
	// WhiteColor is the white color for the cli.
	WhiteColor = lipgloss.Color("#FFFFFF")
	// BlackColor is the black color for the cli.
	BlackColor = lipgloss.Color("#000000")
)

// Styles
var (
	// Header is the style to use for headers
	Header = lipgloss.NewStyle().
		Bold(true).
		Background(AccentColor).
		Foreground(WhiteColor).
		PaddingTop(1).
		PaddingBottom(1).
		PaddingLeft(4).
		PaddingRight(4).
		MaxWidth(80)
	// WelcomeBanner is the style to use for the welcome banner
	WelcomeBanner = lipgloss.NewStyle().
			Bold(true).
			Background(AccentColor).
			Foreground(WhiteColor).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(4).
			PaddingRight(4).
			Width(70)
	// Table is the style to use for tables
	Table = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(AccentColor)
	// TableStyles is the style to use for tables
	TableStyles = table.Styles{
		Selected: lipgloss.NewStyle().Bold(true).Foreground(SecondaryColor),
		Header:   lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(PrimaryColor),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
	}
	// TableHiddenSelectStyles is the style to use for tables. It hides the selection
	// indicator.
	TableHiddenSelectStyles = table.Styles{
		Header:   lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(PrimaryColor),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
		Selected: lipgloss.NewStyle(),
	}
)

// Utility functions
var (
	// HeaderText returns a header with the given text
	HeaderText = Header.Render
	// WelcomeBannerText returns a welcome banner with the given text
	WelcomeBannerText = WelcomeBanner.Render
)

// TableRender renders a table given a table model
func TableRender(t table.Model) string {
	return Table.Render(t.View())
}
