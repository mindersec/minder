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

// Package messages contains messages structs and builders for events
// handled by reconcilers.
package messages

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RepoReconcilerEvent is an event that is sent to the reconciler topic
type RepoReconcilerEvent struct {
	// Project is the project that the event is relevant to
	Project uuid.UUID `json:"project"`
	// Provider is the provider that the event is relevant to
	Provider uuid.UUID `json:"provider"`
	// EntityID is the entity id of the repository to be reconciled
	EntityID uuid.UUID `json:"entity_id"`
}

// NewRepoReconcilerMessage creates a new repos init event
func NewRepoReconcilerMessage(providerID uuid.UUID, entityID uuid.UUID, projectID uuid.UUID) (*message.Message, error) {
	evt := &RepoReconcilerEvent{
		Project:  projectID,
		Provider: providerID,
		EntityID: entityID,
	}

	evtStr, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("error marshalling init event: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), evtStr)
	return msg, nil
}

// CoreContext contains information necessary to further process
// events inside Minder Core.
type CoreContext struct {
	ProviderID uuid.UUID
	ProjectID  uuid.UUID
	Type       string
	Payload    []byte
}

// MinderEvent encapsulate necessary information about creation or
// deletion of entities so that they can be further processed by
// Minder core.
//
// This struct is meant to be used with providers that can push events
// to Minder, or with providers that Minder can poll.
type MinderEvent struct {
	ProviderID uuid.UUID       `json:"provider_id" validate:"required"`
	ProjectID  uuid.UUID       `json:"project_id" validate:"required"`
	EntityType minderv1.Entity `json:"entity_type" validate:"required"`
	EntityID   uuid.UUID       `json:"entity_id"`
	Properties map[string]any  `json:"entity" validate:"required"`
}

// NewMinderEvent creates a new entity added event.
func NewMinderEvent() *MinderEvent {
	return &MinderEvent{
		Properties: map[string]any{},
	}
}

// WithProviderID adds provider id to MinderEvent.
func (e *MinderEvent) WithProviderID(providerID uuid.UUID) *MinderEvent {
	e.ProviderID = providerID
	return e
}

// WithProjectID adds project id to MinderEvent.
func (e *MinderEvent) WithProjectID(projectID uuid.UUID) *MinderEvent {
	e.ProjectID = projectID
	return e
}

// WithProperties adds properties to MinderEvent.
func (e *MinderEvent) WithProperties(props *properties.Properties) *MinderEvent {
	e.Properties = props.ToProtoStruct().AsMap()
	return e
}

// WithEntityID sets the id of the entity.
func (e *MinderEvent) WithEntityID(entityID uuid.UUID) *MinderEvent {
	e.EntityID = entityID
	return e
}

// WithEntityType sets the type of the entity. Type of the entity must
// be meaningful to the Provider.
func (e *MinderEvent) WithEntityType(entityType minderv1.Entity) *MinderEvent {
	e.EntityType = entityType
	return e
}

// ToMessage implements an interface that is currently used on the
// webhook handler. Such interface works by modifiying an existing
// message by means of side effect, which is unnecessary for this
// struct, thus its simplicity.
func (e *MinderEvent) ToMessage(msg *message.Message) error {
	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error marshalling event: %w", err)
	}

	msg.Payload = payload
	msg.Metadata.Set("providerID", e.ProviderID.String())
	msg.Metadata.Set("projectID", e.ProjectID.String())

	return nil
}
