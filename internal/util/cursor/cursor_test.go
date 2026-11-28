// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cursor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:      "non-empty string",
			input:     "testCursor",
			expectErr: false,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: false,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded := EncodeValue(tc.input)
			decoded, err := DecodeValue(encoded)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.input, decoded)
			}
		})
	}
}
