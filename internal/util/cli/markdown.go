// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// RenderMarkdown renders the given string as markdown.
func RenderMarkdown(payload string, opts ...glamour.TermRendererOption) string {
	style := styles.NoTTYStyleConfig
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if termenv.HasDarkBackground() {
			style = styles.DarkStyleConfig
		} else {
			style = styles.LightStyleConfig
		}
	}
	// Remove extra margins from rendering, we can add them in the table
	// if we want.
	style.Document.Margin = nil
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""

	allOpts := []glamour.TermRendererOption{
		glamour.WithStyles(style),
		glamour.WithEmoji(),
	}
	allOpts = append(allOpts, opts...)

	r, err := glamour.NewTermRenderer(
		allOpts...,
	)
	// We can't really fail here, but in case we just return the
	// payload as-is.
	if err != nil {
		return payload
	}
	rendered, err := r.Render(payload)
	// We don't want to fail rendering when input is not valid
	// markdown, and we just output it as-is instead.
	if err != nil {
		return payload
	}
	return rendered
}

// WidthFraction sets the width of the markdown text to the fraction
// of the terminal width (0.0 to 1.0).
func WidthFraction(fraction float64) glamour.TermRendererOption {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w == 0 {
		w = 80 // Default width if we can't determine terminal size
	}
	return glamour.WithWordWrap(int(float64(w) * fraction))
}
