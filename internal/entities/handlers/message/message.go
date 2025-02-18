// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package message contains the message creation strategies
package message

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
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
	// Hint is used to help the entity handler find the entity upstream
	// using the property service. A typical use case is to use the provider
	// in the hint and an upstream ID in the Entity.GetByProps attribute
	Hint EntityHint `json:"hint"`
	// MatchProps is used to match the properties of the found entity. One
	// use-case is to include the hook ID in the MatchProps to match against
	// the entity's hook ID to avoid forwading the message to the wrong entity.
	MatchProps map[string]any `json:"match_props"`
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

// WithMatchProps sets the properties that must match the properties from the found entity
// in order to perform the action.
func (e *HandleEntityAndDoMessage) WithMatchProps(matchProps *properties.Properties) *HandleEntityAndDoMessage {
	e.MatchProps = matchProps.ToProtoStruct().AsMap()
	return e
}
