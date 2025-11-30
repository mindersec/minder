// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		opts     []glamour.TermRendererOption
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:  "normal",
			input: "foo",
			// Output is padded to 80 characters by default
			expected: fmt.Sprintf("foo%s\n", strings.Repeat(" ", 77)),
		},
		{
			name:     "html tags",
			input:    "<div>foo</div>",
			expected: "foo",
		},
		{
			name:     "xss",
			input:    "Hello <STYLE>.XSS{background-image:url(\"javascript:alert('XSS')\");}</STYLE><A CLASS=XSS></A>World",
			expected: "Hello .XSS{background-image:url(\"javascript:alert('XSS')\");}World               \n",
		},
		{
			name:     "script",
			input:    "<script>alert`1`</script>",
			expected: "",
		},
		{
			name:     "div script",
			input:    "<div> <script>alert`1`</script> </div>",
			expected: "",
		},
		{
			name: "multiline",
			input: `<script>alert('$varUnsafe’)</script>
<script>x=’$varUnsafe’</script>
<div onmouseover="'$varUnsafe'"</div>
`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := RenderMarkdown(tt.input)
			require.Equal(t, tt.expected, res)
		})
	}
}
