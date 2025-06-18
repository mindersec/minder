// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzRenderMarkdown(f *testing.F) {
	f.Fuzz(func(t *testing.T, input string) {

		input = "# Header\n" + input

		output := RenderMarkdown(input)
		output = strings.TrimSpace(output)
		require.Contains(t, output, "Header", "Expected output %q to contain Header, got: %q", output, input)
	})
}
