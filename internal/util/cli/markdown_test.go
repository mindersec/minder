//
// Copyright 2024 Stacklok, Inc.
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

package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "\n\n",
		},
		{
			name:  "normal",
			input: "foo",
			// Output is padded to 80 characters by default
			expected: fmt.Sprintf("\n  foo %s \n\n", strings.Repeat(" ", 75)),
		},
		{
			name:     "html tags",
			input:    "<div>foo</div>",
			expected: "\n\n",
		},
		{
			name:     "xss",
			input:    "Hello <STYLE>.XSS{background-image:url(\"javascript:alert('XSS')\");}</STYLE><A CLASS=XSS></A>World",
			expected: "\n  Hello .XSS{background-image:url(\"javascript:alert('XSS')\");}World               \n\n",
		},
		{
			name:     "script",
			input:    "<script>alert`1`</script>",
			expected: "\n\n",
		},
		{
			name:     "div script",
			input:    "<div> <script>alert`1`</script> </div>",
			expected: "\n\n",
		},
		{
			name: "multiline",
			input: `<script>alert('$varUnsafe’)</script>
<script>x=’$varUnsafe’</script>
<div onmouseover="'$varUnsafe'"</div>
`,
			expected: "\n\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := RenderMarkdown(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, res)
		})
	}
}
