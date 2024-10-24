// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzRenderMarkdown(f *testing.F) {
	f.Fuzz(func(t *testing.T, input string) {
		_, err := RenderMarkdown(input)
		require.NoError(t, err)
	})
}
