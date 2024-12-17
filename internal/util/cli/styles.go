// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"math"
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Color Palette
var (
	// PrimaryColor is the primary color for the cli.
	PrimaryColor = lipgloss.Color("#00BBBE")
	// SecondaryColor is the secondary color for the cli.
	SecondaryColor = lipgloss.Color("#59CFA8")
	// AccentColor is the accent color for the cli.
	AccentColor = lipgloss.Color("#3D34E0")
	// WhiteColor is the white color for the cli.
	WhiteColor = lipgloss.Color("#FFFFFF")
	// BlackColor is the black color for the cli.
	BlackColor = lipgloss.Color("#000000")
)

// Common styles
var (
	CursorStyle     = lipgloss.NewStyle().Foreground(SecondaryColor)
	SimpleBoldStyle = lipgloss.NewStyle().Bold(true)
)

// Banner styles
var (
	// DefaultBannerWidth is the default width for a banner
	DefaultBannerWidth = 80
	// Header is the style to use for headers
	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		PaddingTop(1).
		PaddingBottom(1).
		PaddingLeft(1).
		PaddingRight(1).
		MaxWidth(80)
	WarningBanner = lipgloss.NewStyle().
			Bold(true).
			Background(BlackColor).
			Foreground(WhiteColor).
			BorderForeground(AccentColor).
			PaddingTop(2).
			PaddingBottom(2).
			PaddingLeft(4).
			PaddingRight(4).
			Width(DefaultBannerWidth)
	// SuccessBanner is the style to use for a success banner
	SuccessBanner = lipgloss.NewStyle().
			Bold(true).
			Background(AccentColor).
			Foreground(WhiteColor).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(4).
			PaddingRight(4).
			Width(DefaultBannerWidth)
)

func init() {
	// Get the terminal width, if available, and set widths based on terminal width
	fd := os.Stdout.Fd()
	if fd > math.MaxInt32 {
		return
	}
	// checked for overflow explicitly
	// nolint: gosec
	w, _, err := term.GetSize(int(fd))
	if err == nil {
		DefaultBannerWidth = w
	}
}
