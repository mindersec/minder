// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/entities"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// TelemetryStoreWMMiddleware is a Watermill middleware that
// logs the relevant telemetry data.
type TelemetryStoreWMMiddleware struct {
	l *zerolog.Logger
}

// NewTelemetryStoreWMMiddleware returns a new TelemetryStoreWMMiddleware.
func NewTelemetryStoreWMMiddleware(l *zerolog.Logger) *TelemetryStoreWMMiddleware {
	return &TelemetryStoreWMMiddleware{l: l}
}

// TelemetryStoreMiddleware is a Watermill middleware that
// logs the relevant telemetry data.
func (m *TelemetryStoreWMMiddleware) TelemetryStoreMiddleware(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		inf, err := entities.ParseEntityEvent(msg)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling payload: %w", err)
		}

		// Create a new telemetry store from entity
		ts, err := newTelemetryStoreFromEntity(inf)
		if err != nil {
			// Log the error but don't fail the event processing, use the returned empty telemetry store instead
			logger := zerolog.Ctx(msg.Context())
			logger.Info().Msg("error creating telemetry store from entity")
		}

		// Store telemetry data in the context
		ctx := ts.WithTelemetry(msg.Context())
		msg.SetContext(ctx)

		msgs, err := h(msg)

		// Record telemetry
		logMsg := m.l.Info()
		if err != nil {
			logMsg = m.l.Error()
		}
		ts.Record(logMsg).Send()

		return msgs, err
	}
}

// newTelemetryStoreFromEntity creates a new telemetry store from an entity.
func newTelemetryStoreFromEntity(inf *entities.EntityInfoWrapper) (*TelemetryStore, error) {
	// Create a new telemetry store
	ts := &TelemetryStore{}

	// Get the entity UUID - this is the entity we are processing
	ent, err := getEntityID(inf)
	if err != nil {
		// Return an error but also return the telemetry store so we don't fail the event
		return ts, fmt.Errorf("error getting entity ID: %w", err)
	}

	// Set the provider name and project ID
	ts.Provider = inf.Provider
	ts.ProviderID = inf.ProviderID
	ts.Project = inf.ProjectID

	// Set the entity telemetry field based on the entity type
	switch inf.Type {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		ts.Repository = ent
	case minderv1.Entity_ENTITY_ARTIFACTS:
		ts.Artifact = ent
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		ts.PullRequest = ent
	case minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS:
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		// Do nothing
	}

	return ts, nil
}

// getEntityID returns the entity ID from the entity info wrapper based on its type.
func getEntityID(inf *entities.EntityInfoWrapper) (uuid.UUID, error) {
	repoID, artID, prID := inf.GetEntityDBIDs()

	var ent uuid.UUID

	// In the case of this middleware, we receive entities
	// to process by the executor.
	switch inf.Type {
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		return uuid.Nil, fmt.Errorf("unspecified entity type")
	case minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return uuid.Nil, fmt.Errorf("build environments not supported")
	case minderv1.Entity_ENTITY_REPOSITORIES:
		ent = repoID.UUID
	case minderv1.Entity_ENTITY_ARTIFACTS:
		ent = artID.UUID
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		ent = prID.UUID
	}

	return ent, nil
}
