// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/cli/table/layouts"
)

//nolint:paralleltest // Each case sets the terminal width through the environment.
func TestRenderPreservesLineBreaks(t *testing.T) {
	tests := []struct {
		name  string
		width string
		input string
		want  []string
	}{
		{
			name:  "nested YAML",
			width: "40",
			input: "event:\n  - workflow_dispatch\nrunner: hosted",
			want:  []string{"event:", "  - workflow_dispatch", "runner: hosted"},
		},
		{
			name:  "soft wrapping within each line",
			width: "20",
			input: "first line has several words\nsecond: value",
			want:  []string{"first line has", "several words", "second: value"},
		},
		{
			name:  "blank line",
			width: "20",
			input: "first: value\n\nsecond: value",
			want:  []string{"first: value", "", "second: value"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("COLUMNS", tt.width)
			for _, colored := range []bool{false, true} {
				var out bytes.Buffer
				table := New(Simple, layouts.Default, &out, []string{"Definition"})
				if colored {
					table.AddRowWithColor(layouts.NoColor(tt.input))
				} else {
					table.AddRow(tt.input)
				}
				table.Render()

				lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")[2:]
				for i, line := range lines {
					lines[i] = strings.TrimRight(strings.TrimPrefix(line, " "), " ")
				}
				require.Equal(t, tt.want, lines)
			}
		})
	}
}

func TestWrapLinesPreservesColors(t *testing.T) {
	t.Parallel()
	for _, color := range []text.Color{text.FgRed, text.FgGreen, text.FgYellow} {
		start := color.EscapeSeq()
		input := start + "first line has several words\nsecond: value" + text.EscapeReset
		wrapped := wrapLines(input, 18)
		require.Equal(t, "first line has    \nseveral words\nsecond: value", text.StripEscape(wrapped))
		require.Contains(t, wrapped, "\n"+start+"second: value"+text.EscapeReset)

		input = start + "first line has several words" + text.EscapeReset + "\nplain"
		require.True(t, strings.HasSuffix(wrapLines(input, 18), "\nplain"))
	}
}
