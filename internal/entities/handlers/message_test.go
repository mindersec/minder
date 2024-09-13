package handlers

import (
	"testing"

	"github.com/stacklok/minder/internal/entities/properties"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityRefreshAndDoMessageRoundTrip(t *testing.T) {
	scenarios := []struct {
		name           string
		props          map[string]any
		entType        v1.Entity
		nextHandler    string
		providerHint   string
		expectedError  bool
		errorAssertion func(t *testing.T, err error)
	}{
		{
			name: "Valid repository entity",
			props: map[string]any{
				"id":   "123",
				"name": "test-repo",
			},
			entType:      v1.Entity_ENTITY_REPOSITORIES,
			nextHandler:  "next-handler",
			providerHint: "github",
		},
		{
			name: "Valid artifact entity",
			props: map[string]any{
				"id":      "456",
				"version": "1.0.0",
			},
			entType:      v1.Entity_ENTITY_ARTIFACTS,
			nextHandler:  "artifact-handler",
			providerHint: "docker",
		},
		{
			name: "Missing next handler",
			props: map[string]any{
				"id": "789",
			},
			entType:       v1.Entity_ENTITY_PULL_REQUESTS,
			nextHandler:   "",
			providerHint:  "gitlab",
			expectedError: true,
			errorAssertion: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "missing next handler name")
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// Create Properties
			props, err := properties.NewProperties(sc.props)
			require.NoError(t, err)

			// Create HandleEntityAndDoMessage
			original := NewEntityRefreshAndDoMessage(sc.entType, props, sc.nextHandler, sc.providerHint)

			// Convert to watermill Message
			msg, err := original.ToMessage()
			require.NoError(t, err)

			// Convert back to HandleEntityAndDoMessage
			roundTrip, err := messageToEntityRefreshAndDo(msg)

			if sc.expectedError {
				assert.Error(t, err)
				if sc.errorAssertion != nil {
					sc.errorAssertion(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, original.GetByProps, roundTrip.GetByProps)
				assert.Equal(t, original.EntType, roundTrip.EntType)
				assert.Equal(t, original.NextHandlerName, roundTrip.NextHandlerName)
				assert.Equal(t, original.ProviderHint, roundTrip.ProviderHint)
			}
		})
	}
}
