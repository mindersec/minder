// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package message

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestEntityRefreshAndDoMessageRoundTrip(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		props         map[string]any
		entType       v1.Entity
		ownerProps    map[string]any
		matchProps    map[string]any
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
		{
			name: "Entity with matching properties",
			props: map[string]any{
				"id": "123",
			},
			matchProps: map[string]any{
				"id": "456",
			},
			entType:       v1.Entity_ENTITY_REPOSITORIES,
			providerHint:  "github",
			providerClass: string(db.ProviderClassGithub),
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()

			props := properties.NewProperties(sc.props)

			original := NewEntityRefreshAndDoMessage().
				WithEntity(sc.entType, props).
				WithProviderImplementsHint(sc.providerHint).
				WithProviderClassHint(sc.providerClass)

			if sc.ownerProps != nil {
				ownerProps := properties.NewProperties(sc.ownerProps)
				original.WithOriginator(sc.ownerType, ownerProps)
			}

			if sc.matchProps != nil {
				matchProps := properties.NewProperties(sc.matchProps)
				original.WithMatchProps(matchProps)
			}

			handlerMsg := message.NewMessage(uuid.New().String(), nil)
			err := original.ToMessage(handlerMsg)
			require.NoError(t, err)

			roundTrip, err := ToEntityRefreshAndDo(handlerMsg)
			require.NoError(t, err)
			assert.Equal(t, original.Entity.GetByProps, roundTrip.Entity.GetByProps)
			assert.Equal(t, original.Entity.Type, roundTrip.Entity.Type)
			assert.Equal(t, original.Hint.ProviderImplementsHint, roundTrip.Hint.ProviderImplementsHint)
			assert.Equal(t, original.Hint.ProviderClassHint, roundTrip.Hint.ProviderClassHint)
			if original.Originator.Type != v1.Entity_ENTITY_UNSPECIFIED {
				assert.Equal(t, original.Originator.GetByProps, roundTrip.Originator.GetByProps)
				assert.Equal(t, original.Originator.Type, roundTrip.Originator.Type)
			}
			if original.MatchProps != nil {
				assert.Equal(t, original.MatchProps, roundTrip.MatchProps)
			}
		})
	}
}
