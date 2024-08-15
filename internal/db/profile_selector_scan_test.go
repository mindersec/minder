// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package db

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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
			input: []byte(fmt.Sprintf("(%s,%s,repository,\"entity.name == \"\"test/test\"\" && repository.is_fork != true\",\"comment1\")", selectorId, profileId)),
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
