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

package message

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/entities/properties"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestEntityRefreshAndDoMessageRoundTrip(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		props         map[string]any
		entType       v1.Entity
		ownerProps    map[string]any
		ownerType     v1.Entity
		providerHint  string
		providerClass string
	}{
		{
			name: "Valid repository entity",
			props: map[string]any{
				"id":   "123",
				"name": "test-repo",
			},
			entType:       v1.Entity_ENTITY_REPOSITORIES,
			providerHint:  "github",
			providerClass: string(db.ProviderClassGithub),
		},
		{
			name: "Valid artifact entity",
			props: map[string]any{
				"id":      "456",
				"version": "1.0.0",
			},
			entType: v1.Entity_ENTITY_ARTIFACTS,
			ownerProps: map[string]any{
				"id": "123",
			},
			ownerType:     v1.Entity_ENTITY_REPOSITORIES,
			providerHint:  "docker",
			providerClass: string(db.ProviderClassDockerhub),
		},
		{
			name: "Valid pull request entity",
			props: map[string]any{
				"id": "789",
			},
			entType: v1.Entity_ENTITY_PULL_REQUESTS,
			ownerProps: map[string]any{
				"id": "123",
			},
			ownerType:     v1.Entity_ENTITY_REPOSITORIES,
			providerHint:  "github",
			providerClass: string(db.ProviderClassGithub),
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()

			props, err := properties.NewProperties(sc.props)
			require.NoError(t, err)

			original := NewEntityRefreshAndDoMessage().
				WithEntity(sc.entType, props).
				WithProviderImplementsHint(sc.providerHint).
				WithProviderClassHint(sc.providerClass)

			if sc.ownerProps != nil {
				ownerProps, err := properties.NewProperties(sc.ownerProps)
				require.NoError(t, err)
				original.WithOwner(sc.ownerType, ownerProps)
			}

			handlerMsg := message.NewMessage(uuid.New().String(), nil)
			err = original.ToMessage(handlerMsg)
			require.NoError(t, err)

			roundTrip, err := ToEntityRefreshAndDo(handlerMsg)
			assert.NoError(t, err)
			assert.Equal(t, original.Entity.GetByProps, roundTrip.Entity.GetByProps)
			assert.Equal(t, original.Entity.Type, roundTrip.Entity.Type)
			assert.Equal(t, original.Hint.ProviderImplementsHint, roundTrip.Hint.ProviderImplementsHint)
			assert.Equal(t, original.Hint.ProviderClassHint, roundTrip.Hint.ProviderClassHint)
			if original.Originator.Type != v1.Entity_ENTITY_UNSPECIFIED {
				assert.Equal(t, original.Originator.GetByProps, roundTrip.Originator.GetByProps)
				assert.Equal(t, original.Originator.Type, roundTrip.Originator.Type)
			}
		})
	}
}
