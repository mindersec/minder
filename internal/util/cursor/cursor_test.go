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

func TestEncodeDecodeCursor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		repoCursor        RepoCursor
		expectEmptyCursor bool
	}{
		{
			name:              "all inputs valid",
			repoCursor:        RepoCursor{ProjectId: "3b241101-e2bb-4255-8caf-4136c566a964", Provider: "testProvider", RepoId: 123},
			expectEmptyCursor: false,
		},
		{
			name:              "empty projectId",
			repoCursor:        RepoCursor{ProjectId: "", Provider: "testProvider", RepoId: 123},
			expectEmptyCursor: true,
		},
		{
			name:              "empty provider",
			repoCursor:        RepoCursor{ProjectId: "3b241101-e2bb-4255-8caf-4136c566a964", Provider: "", RepoId: 123},
			expectEmptyCursor: true,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encodedCursor := tc.repoCursor.String()

			if tc.expectEmptyCursor {
				require.Equal(t, "", encodedCursor)
			} else {
				decodedRepoCursor, err := NewRepoCursor(encodedCursor)

				require.NoError(t, err)
				require.Equal(t, tc.repoCursor.ProjectId, decodedRepoCursor.ProjectId)
				require.Equal(t, tc.repoCursor.Provider, decodedRepoCursor.Provider)
				require.Equal(t, tc.repoCursor.RepoId, decodedRepoCursor.RepoId)
			}
		})
	}
}

func TestDecodeListRepositoriesByProjectIDCursor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		cursor              string
		expectedRepoCursor  *RepoCursor
		expectErrorDecoding bool
	}{
		{
			name:                "compliant input",
			cursor:              EncodeValue("3b241101-e2bb-4255-8caf-4136c566a964,testProvider,123"),
			expectedRepoCursor:  &RepoCursor{ProjectId: "3b241101-e2bb-4255-8caf-4136c566a964", Provider: "testProvider", RepoId: 123},
			expectErrorDecoding: false,
		},
		{
			name:                "non-compliant input",
			cursor:              EncodeValue("nonCompliantInput"),
			expectedRepoCursor:  nil,
			expectErrorDecoding: true,
		},
		{
			name:                "non-compliant 64 bit input",
			cursor:              EncodeValue("3b241101-e2bb-4255-8caf-4136c566a964,testProvider,12345678901234567890"),
			expectedRepoCursor:  nil,
			expectErrorDecoding: true,
		},
		{
			name:                "empty input",
			cursor:              "",
			expectedRepoCursor:  &RepoCursor{},
			expectErrorDecoding: false,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decodedRepoCursor, err := NewRepoCursor(tc.cursor)

			if tc.expectErrorDecoding {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedRepoCursor.ProjectId, decodedRepoCursor.ProjectId)
				require.Equal(t, tc.expectedRepoCursor.Provider, decodedRepoCursor.Provider)
				require.Equal(t, tc.expectedRepoCursor.RepoId, decodedRepoCursor.RepoId)
			}
		})
	}
}
