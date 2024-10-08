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

// Package message contains the message creation strategies
package message

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/entities/properties"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// TypedProps is a struct that contains the type of entity and its properties.
// it is used for either the entity or the owner entity.
type TypedProps struct {
	Type       v1.Entity      `json:"type"`
	EntityID   uuid.UUID      `json:"entity_id"`
	GetByProps map[string]any `json:"get_by_props"`
}

// EntityHint is a hint that is used to help the entity handler find the entity.
type EntityHint struct {
	ProviderImplementsHint string `json:"provider_implements_hint"`
	ProviderClassHint      string `json:"provider_class_hint"`
}

// HandleEntityAndDoMessage is a message that is sent to the entity handler to refresh an entity and perform an action.
type HandleEntityAndDoMessage struct {
	Entity     TypedProps `json:"entity"`
	Originator TypedProps `json:"owner"`
	Hint       EntityHint `json:"hint"`
}

// NewEntityRefreshAndDoMessage creates a new HandleEntityAndDoMessage struct.
func NewEntityRefreshAndDoMessage() *HandleEntityAndDoMessage {
	return &HandleEntityAndDoMessage{}
}

// WithEntity sets the entity and its properties.
func (e *HandleEntityAndDoMessage) WithEntity(entType v1.Entity, getByProps *properties.Properties) *HandleEntityAndDoMessage {
	e.Entity = TypedProps{
		Type:       entType,
		GetByProps: getByProps.ToProtoStruct().AsMap(),
	}
	return e
}

// WithOriginator sets the owner entity and its properties.
func (e *HandleEntityAndDoMessage) WithOriginator(
	originatorType v1.Entity, originatorProps *properties.Properties,
) *HandleEntityAndDoMessage {
	e.Originator = TypedProps{
		Type:       originatorType,
		GetByProps: originatorProps.ToProtoStruct().AsMap(),
	}
	return e
}

// WithEntityID sets the entity ID for the entity that will be used when looking up the entity.
func (e *HandleEntityAndDoMessage) WithEntityID(entityID uuid.UUID) *HandleEntityAndDoMessage {
	e.Entity.EntityID = entityID
	return e
}

// WithProviderImplementsHint sets the provider hint for the entity that will be used when looking up the entity.
// to the provider implements hint
func (e *HandleEntityAndDoMessage) WithProviderImplementsHint(providerHint string) *HandleEntityAndDoMessage {
	e.Hint.ProviderImplementsHint = providerHint
	return e
}

// WithProviderClassHint sets the provider hint for the entity that will be used when looking up the entity.
// to the provider class
func (e *HandleEntityAndDoMessage) WithProviderClassHint(providerClassHint string) *HandleEntityAndDoMessage {
	e.Hint.ProviderClassHint = providerClassHint
	return e
}

// ToEntityRefreshAndDo converts a Watermill message to a HandleEntityAndDoMessage struct.
func ToEntityRefreshAndDo(msg *message.Message) (*HandleEntityAndDoMessage, error) {
	entMsg := &HandleEntityAndDoMessage{}

	err := json.Unmarshal(msg.Payload, entMsg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling entity: %w", err)
	}

	return entMsg, nil
}

// ToMessage converts the HandleEntityAndDoMessage struct to a Watermill message.
func (e *HandleEntityAndDoMessage) ToMessage(msg *message.Message) error {
	payloadBytes, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error marshalling entity: %w", err)
	}

	msg.Payload = payloadBytes
	return nil
}
