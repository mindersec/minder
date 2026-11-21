// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestNewRepoReconcilerMessage(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()
	entityID := uuid.New()
	projectID := uuid.New()

	scenarios := []struct {
		name       string
		providerID uuid.UUID
		entityID   uuid.UUID
		projectID  uuid.UUID
	}{
		{
			name:       "valid repo reconciler event message",
			providerID: providerID,
			entityID:   entityID,
			projectID:  projectID,
		},
		{
			name:       "handles nil uuid fields gracefully",
			providerID: uuid.Nil,
			entityID:   uuid.Nil,
			projectID:  uuid.Nil,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg, err := NewRepoReconcilerMessage(tt.providerID, tt.entityID, tt.projectID)
			require.NoError(t, err)
			require.NotNil(t, msg)

			_, parseErr := uuid.Parse(msg.UUID)
			require.NoError(t, parseErr)

			var evt RepoReconcilerEvent
			require.NoError(t, json.Unmarshal(msg.Payload, &evt))
			require.Equal(t, tt.providerID, evt.Provider)
			require.Equal(t, tt.entityID, evt.EntityID)
			require.Equal(t, tt.projectID, evt.Project)
		})
	}
}

func TestMinderEvent_BuilderPattern(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()
	projectID := uuid.New()
	entityID := uuid.New()

	props := properties.NewProperties(map[string]any{
		"repo_name": "stacklok/minder",
		"is_fork":   false,
		"stars":     1200,
	})

	scenarios := []struct {
		name            string
		buildEvent      func() *MinderEvent
		expectedProps   map[string]any
		expectedProvID  uuid.UUID
		expectedProjID  uuid.UUID
		expectedEntID   uuid.UUID
		expectedEntType minderv1.Entity
	}{
		{
			name: "default initialized event has non-nil empty properties map",
			buildEvent: func() *MinderEvent {
				return NewMinderEvent()
			},
			expectedProps:   map[string]any{},
			expectedProvID:  uuid.Nil,
			expectedProjID:  uuid.Nil,
			expectedEntID:   uuid.Nil,
			expectedEntType: minderv1.Entity_ENTITY_UNSPECIFIED,
		},
		{
			name: "full builder chain populates all event fields accurately",
			buildEvent: func() *MinderEvent {
				return NewMinderEvent().
					WithProviderID(providerID).
					WithProjectID(projectID).
					WithEntityID(entityID).
					WithEntityType(minderv1.Entity_ENTITY_REPOSITORIES).
					WithProperties(props)
			},
			expectedProps: map[string]any{
				"repo_name": "stacklok/minder",
				"is_fork":   false,
				"stars":     float64(1200),
			},
			expectedProvID:  providerID,
			expectedProjID:  projectID,
			expectedEntID:   entityID,
			expectedEntType: minderv1.Entity_ENTITY_REPOSITORIES,
		},
		{
			name: "builder chain allows incremental mutation",
			buildEvent: func() *MinderEvent {
				evt := NewMinderEvent()
				evt.WithProviderID(providerID)
				evt.WithProjectID(projectID)
				evt.WithEntityType(minderv1.Entity_ENTITY_PULL_REQUESTS)
				return evt
			},
			expectedProps:   map[string]any{},
			expectedProvID:  providerID,
			expectedProjID:  projectID,
			expectedEntID:   uuid.Nil,
			expectedEntType: minderv1.Entity_ENTITY_PULL_REQUESTS,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evt := tt.buildEvent()
			require.NotNil(t, evt)
			require.Equal(t, tt.expectedProvID, evt.ProviderID)
			require.Equal(t, tt.expectedProjID, evt.ProjectID)
			require.Equal(t, tt.expectedEntID, evt.EntityID)
			require.Equal(t, tt.expectedEntType, evt.EntityType)
			require.Equal(t, tt.expectedProps, evt.Properties)
		})
	}
}

func TestMinderEvent_ToMessage(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()
	projectID := uuid.New()
	entityID := uuid.New()

	props := properties.NewProperties(map[string]any{
		"artifact_name": "minder-container",
		"tags":          []any{"v1.0.0", "latest"},
	})

	scenarios := []struct {
		name       string
		event      *MinderEvent
		initialMsg *message.Message
	}{
		{
			name: "serializes populated event into message payload and sets metadata headers",
			event: NewMinderEvent().
				WithProviderID(providerID).
				WithProjectID(projectID).
				WithEntityID(entityID).
				WithEntityType(minderv1.Entity_ENTITY_ARTIFACTS).
				WithProperties(props),
			initialMsg: message.NewMessage(uuid.New().String(), nil),
		},
		{
			name:       "serializes empty event and sets empty metadata headers correctly",
			event:      NewMinderEvent(),
			initialMsg: message.NewMessage(uuid.New().String(), []byte("prior-data")),
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.event.ToMessage(tt.initialMsg)
			require.NoError(t, err)

			require.Equal(t, tt.event.ProviderID.String(), tt.initialMsg.Metadata.Get("providerID"))
			require.Equal(t, tt.event.ProjectID.String(), tt.initialMsg.Metadata.Get("projectID"))

			var decoded MinderEvent
			require.NoError(t, json.Unmarshal(tt.initialMsg.Payload, &decoded))
			require.Equal(t, tt.event.ProviderID, decoded.ProviderID)
			require.Equal(t, tt.event.ProjectID, decoded.ProjectID)
			require.Equal(t, tt.event.EntityID, decoded.EntityID)
			require.Equal(t, tt.event.EntityType, decoded.EntityType)
			require.Equal(t, tt.event.Properties, decoded.Properties)
		})
	}
}
