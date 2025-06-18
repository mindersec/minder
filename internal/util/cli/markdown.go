// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders the given string as markdown.
func RenderMarkdown(payload string) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
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
