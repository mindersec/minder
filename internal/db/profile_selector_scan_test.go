// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestScan tests the Scan method of the ProfileSelector struct
// for more tests that exercise the retrieval of a profile with selectors that also happens to use Scan
// see TestProfileListWithSelectors
func TestScan(t *testing.T) {
	t.Parallel()

	selectorId := uuid.New()
	profileId := uuid.New()

	tc := []struct {
		name     string
		input    interface{}
		expected ProfileSelector
	}{
		{
			name:  "Valid input with all fields",
			input: []byte(fmt.Sprintf(`(%s,%s,repository,"entity.name == ""test/test"" && repository.is_fork != true","comment1")`, selectorId, profileId)),
			expected: ProfileSelector{
				ID:        selectorId,
				ProfileID: profileId,
				Entity: NullEntities{
					Valid:    true,
					Entities: EntitiesRepository,
				},
				Selector: "entity.name == \"test/test\" && repository.is_fork != true",
				Comment:  "comment1",
			},
		},
		{
			name:  "Valid input with commas in the selector",
			input: []byte(fmt.Sprintf(`(%s,%s,repository,"repository.properties['github/primary_language'] in ['TypeScript', 'Go']","comment1")`, selectorId, profileId)),
			expected: ProfileSelector{
				ID:        selectorId,
				ProfileID: profileId,
				Entity: NullEntities{
					Valid:    true,
					Entities: EntitiesRepository,
				},
				Selector: "repository.properties['github/primary_language'] in ['TypeScript', 'Go']",
				Comment:  "comment1",
			},
		},
		{
			name:  "Comment includes uneven quotes",
			input: []byte(fmt.Sprintf(`(%s,%s,repository,"repository.name == foo",""comment1")`, selectorId, profileId)),
			expected: ProfileSelector{
				ID:        selectorId,
				ProfileID: profileId,
				Entity: NullEntities{
					Valid:    true,
					Entities: EntitiesRepository,
				},
				Selector: "repository.name == foo",
				Comment:  "comment1",
			},
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var ps ProfileSelector
			err := ps.Scan(tc.input)

			assert.NoError(t, err, "Expected no error")
			assert.Equal(t, tc.expected.ID, ps.ID)
			assert.Equal(t, tc.expected.ProfileID, ps.ProfileID)
			assert.Equal(t, tc.expected.Entity, ps.Entity)
			assert.Equal(t, tc.expected.Selector, ps.Selector)
			assert.Equal(t, tc.expected.Comment, ps.Comment)
		})
	}
}
