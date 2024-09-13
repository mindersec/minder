package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stacklok/minder/internal/entities/properties"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type TypedProps struct {
	Type       v1.Entity      `json:"type"`
	GetByProps map[string]any `json:"get_by_props"`
}

type EntityHint struct {
	ProviderHint  string `json:"provider_hint"`
	PropertyKey   string `json:"property_key"`
	PropertyValue any    `json:"property_value"`
}

type HandleEntityAndDoMessage struct {
	Entity TypedProps `json:"entity"`
	Owner  TypedProps `json:"owner"`
	Hint   EntityHint `json:"hint"`
}

func NewEntityRefreshAndDoMessage() *HandleEntityAndDoMessage {
	return &HandleEntityAndDoMessage{}
}

func (e *HandleEntityAndDoMessage) WithEntity(entType v1.Entity, getByProps *properties.Properties) *HandleEntityAndDoMessage {
	e.Entity = TypedProps{
		Type:       entType,
		GetByProps: getByProps.ToProtoStruct().AsMap(),
	}
	return e
}

func (e *HandleEntityAndDoMessage) WithOwner(ownerType v1.Entity, ownerProps *properties.Properties) *HandleEntityAndDoMessage {
	e.Owner = TypedProps{
		Type:       ownerType,
		GetByProps: ownerProps.ToProtoStruct().AsMap(),
	}
	return e
}

func (e *HandleEntityAndDoMessage) WithProviderHint(providerHint string) *HandleEntityAndDoMessage {
	e.Hint.ProviderHint = providerHint
	return e
}

func (e *HandleEntityAndDoMessage) WithPropertyHint(key string, value any) *HandleEntityAndDoMessage {
	e.Hint.PropertyKey = key
	e.Hint.PropertyValue = value
	return e
}

func messageToEntityRefreshAndDo(msg *message.Message) (*HandleEntityAndDoMessage, error) {
	entMsg := &HandleEntityAndDoMessage{}

	err := json.Unmarshal(msg.Payload, entMsg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling entity: %w", err)
	}

	return entMsg, nil
}

func (e *HandleEntityAndDoMessage) ToMessage(msg *message.Message) error {
	payloadBytes, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error marshalling entity: %w", err)
	}

	msg.Payload = payloadBytes
	return nil
}
