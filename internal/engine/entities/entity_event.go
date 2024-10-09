// Copyright 2023 Stacklok, Inc.
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

package entities

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/events"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// EntityInfoWrapper is a helper struct to gather information
// about entities from events.
// It's able to build message.Message structures from
// the information it gathers.
//
// It's also able to read the message.Message that contains a payload
// with a protobuf message that's specific to the entity type.
//
// It also assumes the following metadata keys are present:
//
// - EntityTypeEventKey - entity_type
// - EntityIDEventKey - entity_id
type EntityInfoWrapper struct {
	ProviderID    uuid.UUID
	ProjectID     uuid.UUID
	EntityID      uuid.UUID
	Entity        protoreflect.ProtoMessage
	Type          minderv1.Entity
	OwnershipData map[string]string
	ExecutionID   *uuid.UUID
	ActionEvent   string
}

const (
	// EntityTypeEventKey is the key for the entity type
	EntityTypeEventKey = "entity_type"
	// EntityIDEventKey is the key for the entity ID
	// Note that we'll be migrating to this key
	// and deprecating the other entity ID keys
	EntityIDEventKey = "entity_id"
	// ProviderIDEventKey is the key for the provider ID
	ProviderIDEventKey = "provider_id"
	// ProjectIDEventKey is the key for the project ID
	ProjectIDEventKey = "project_id"
	// repositoryIDEventKey is the key for the repository ID
	repositoryIDEventKey = "repository_id"
	// artifactIDEventKey is the key for the artifact ID
	artifactIDEventKey = "artifact_id"
	// pullRequestIDEventKey is the key for the pull request ID
	pullRequestIDEventKey = "pull_request_id"
	// ExecutionIDKey is the key for the execution ID. This is set when acquiring a lock.
	ExecutionIDKey = "execution_id"
)

// NewEntityInfoWrapper creates a new EntityInfoWrapper
func NewEntityInfoWrapper() *EntityInfoWrapper {
	return &EntityInfoWrapper{
		OwnershipData: make(map[string]string),
	}
}

// WithProviderID sets the provider ID
func (eiw *EntityInfoWrapper) WithProviderID(providerID uuid.UUID) *EntityInfoWrapper {
	eiw.ProviderID = providerID

	return eiw
}

// WithProtoMessage sets the entity to a protobuf message
// and sets the entity type
func (eiw *EntityInfoWrapper) WithProtoMessage(entType minderv1.Entity, msg protoreflect.ProtoMessage) *EntityInfoWrapper {
	eiw.Type = entType
	eiw.Entity = msg

	return eiw
}

// WithArtifact sets the entity to a versioned artifact sets the entity to a versioned artifact
func (eiw *EntityInfoWrapper) WithArtifact(va *minderv1.Artifact) *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_ARTIFACTS
	eiw.Entity = va

	return eiw
}

// WithRepository sets the entity to a repository
func (eiw *EntityInfoWrapper) WithRepository(r *minderv1.Repository) *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_REPOSITORIES
	eiw.Entity = r

	return eiw
}

// WithPullRequest sets the entity to a repository
func (eiw *EntityInfoWrapper) WithPullRequest(p *minderv1.PullRequest) *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_PULL_REQUESTS
	eiw.Entity = p

	return eiw
}

// WithEntityInstance sets the entity to an entity instance
func (eiw *EntityInfoWrapper) WithEntityInstance(etyp minderv1.Entity, ei *minderv1.EntityInstance) *EntityInfoWrapper {
	eiw.Type = etyp
	eiw.Entity = ei

	return eiw
}

// WithProjectID sets the project ID
func (eiw *EntityInfoWrapper) WithProjectID(id uuid.UUID) *EntityInfoWrapper {
	eiw.ProjectID = id

	return eiw
}

// WithID sets the ID for an entity type
func (eiw *EntityInfoWrapper) WithID(id uuid.UUID) *EntityInfoWrapper {
	eiw.EntityID = id

	return eiw
}

// WithExecutionID sets the execution ID
func (eiw *EntityInfoWrapper) WithExecutionID(id uuid.UUID) *EntityInfoWrapper {
	eiw.ExecutionID = &id

	return eiw
}

// AsRepository sets the entity type to a repository
func (eiw *EntityInfoWrapper) AsRepository() *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_REPOSITORIES
	eiw.Entity = &minderv1.Repository{}

	return eiw
}

// AsArtifact sets the entity type to a versioned artifact
func (eiw *EntityInfoWrapper) AsArtifact() *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_ARTIFACTS
	eiw.Entity = &minderv1.Artifact{}

	return eiw
}

// AsPullRequest sets the entity type to a pull request
func (eiw *EntityInfoWrapper) AsPullRequest() {
	eiw.Type = minderv1.Entity_ENTITY_PULL_REQUESTS
	eiw.Entity = &minderv1.PullRequest{}
}

// AsEntityInstance sets the entity type to an entity instance
func (eiw *EntityInfoWrapper) AsEntityInstance(entityType minderv1.Entity) {
	eiw.Type = entityType
	eiw.Entity = &minderv1.EntityInstance{}
}

// BuildMessage builds a message.Message from the information
func (eiw *EntityInfoWrapper) BuildMessage() (*message.Message, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error generating UUID: %w", err)
	}

	msg := message.NewMessage(id.String(), nil)
	if err := eiw.ToMessage(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// Publish builds a message.Message and publishes it to the event bus
func (eiw *EntityInfoWrapper) Publish(evt events.Publisher) error {
	msg, err := eiw.BuildMessage()
	if err != nil {
		return err
	}

	if err := evt.Publish(events.TopicQueueEntityEvaluate, msg); err != nil {
		return fmt.Errorf("error publishing entity event: %w", err)
	}

	return nil
}

// ToMessage sets the information to a message.Message
func (eiw *EntityInfoWrapper) ToMessage(msg *message.Message) error {
	typ := eiw.Type.ToString()

	if eiw.ProjectID == uuid.Nil {
		return fmt.Errorf("project ID is required")
	}

	if eiw.ProviderID == uuid.Nil {
		return fmt.Errorf("provider ID is required")
	}

	if eiw.EntityID != uuid.Nil {
		msg.Metadata.Set(EntityIDEventKey, eiw.EntityID.String())
	}

	if eiw.ExecutionID != nil {
		msg.Metadata.Set(ExecutionIDKey, eiw.ExecutionID.String())
	}

	msg.Metadata.Set(ProviderIDEventKey, eiw.ProviderID.String())
	msg.Metadata.Set(EntityTypeEventKey, typ)
	msg.Metadata.Set(ProjectIDEventKey, eiw.ProjectID.String())
	for k, v := range eiw.OwnershipData {
		msg.Metadata.Set(k, v)
	}
	var err error
	msg.Payload, err = protojson.Marshal(eiw.Entity)
	if err != nil {
		return fmt.Errorf("error marshalling repository: %w", err)
	}

	return nil
}

// GetID returns the entity ID.
func (eiw *EntityInfoWrapper) GetID() (uuid.UUID, error) {
	if eiw == nil {
		return uuid.Nil, fmt.Errorf("no entity info wrapper")
	}

	if eiw.EntityID != uuid.Nil {
		return eiw.EntityID, nil
	}

	// Fall back to the ownership data
	id, ok := eiw.getIDForEntityType(eiw.Type)
	if ok {
		return id, nil
	}

	return uuid.Nil, fmt.Errorf("no entity ID found")
}

// This will be deprecated in the future in favor of relying on the entity ID key.
// For now, this is just a fallback.
func (eiw *EntityInfoWrapper) getIDForEntityType(t minderv1.Entity) (uuid.UUID, bool) {
	key, err := getEntityMetadataKey(t)
	if err != nil {
		return uuid.Nil, false
	}

	if id, ok := eiw.OwnershipData[key]; ok {
		return uuid.MustParse(id), true
	}

	return uuid.Nil, false
}

func (eiw *EntityInfoWrapper) withProjectIDFromMessage(msg *message.Message) error {
	rawID := msg.Metadata.Get(ProjectIDEventKey)
	if rawID == "" {
		return fmt.Errorf("%s not found in metadata", ProjectIDEventKey)
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("error parsing project ID: %w", err)
	}

	eiw.ProjectID = id
	return nil
}

func (eiw *EntityInfoWrapper) withProviderIDFromMessage(msg *message.Message) error {
	rawProviderID := msg.Metadata.Get(ProviderIDEventKey)
	if rawProviderID == "" {
		return fmt.Errorf("%s not found in metadata", ProviderIDEventKey)
	}

	providerID, err := uuid.Parse(rawProviderID)
	if err != nil {
		return fmt.Errorf("malformed provider id %s", rawProviderID)
	}

	eiw.ProviderID = providerID
	return nil
}

func (eiw *EntityInfoWrapper) withRepositoryIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, repositoryIDEventKey)
}

func (eiw *EntityInfoWrapper) withArtifactIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, artifactIDEventKey)
}

func (eiw *EntityInfoWrapper) withPullRequestIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, pullRequestIDEventKey)
}

func (eiw *EntityInfoWrapper) withEntityInstanceIDFromMessage(msg *message.Message) error {
	rawEntityID := msg.Metadata.Get(EntityIDEventKey)
	if rawEntityID == "" {
		return fmt.Errorf("%s not found in metadata", EntityIDEventKey)
	}

	entityID, err := uuid.Parse(rawEntityID)
	if err != nil {
		return fmt.Errorf("malformed entity id %s", rawEntityID)
	}

	eiw.EntityID = entityID
	return nil
}

// WithExecutionIDFromMessage sets the execution ID from the message
func (eiw *EntityInfoWrapper) WithExecutionIDFromMessage(msg *message.Message) error {
	executionID := msg.Metadata.Get(ExecutionIDKey)
	if executionID == "" {
		return fmt.Errorf("%s not found in metadata", ExecutionIDKey)
	}

	id, err := uuid.Parse(executionID)
	if err != nil {
		return fmt.Errorf("error parsing execution ID: %w", err)
	}

	eiw.ExecutionID = &id
	return nil
}

func (eiw *EntityInfoWrapper) withIDFromMessage(msg *message.Message, key string) error {
	id, err := getIDFromMessage(msg, key)
	if err != nil {
		return fmt.Errorf("error parsing %s: %w", key, err)
	}

	eiw.OwnershipData[key] = id
	return nil
}

func (eiw *EntityInfoWrapper) withID(key string, id string) {
	eiw.OwnershipData[key] = id
}

func (eiw *EntityInfoWrapper) unmarshalEntity(msg *message.Message) error {
	return protojson.Unmarshal(msg.Payload, eiw.Entity)
}

// This is only used to get a specific entity ID from the metadata
// This will be deprecated in the future in favor of relying on the entity ID key
func getEntityMetadataKey(t minderv1.Entity) (string, error) {
	//nolint:exhaustive // We want to fail if it's not one of the explicit types
	switch t {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return repositoryIDEventKey, nil
	case minderv1.Entity_ENTITY_ARTIFACTS:
		return artifactIDEventKey, nil
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return pullRequestIDEventKey, nil
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		return "", fmt.Errorf("entity type unspecified")
	default:
		return "", fmt.Errorf("unknown or unsupported entity type: %s", t.String())
	}
}

func getIDFromMessage(msg *message.Message, key string) (string, error) {
	rawID := msg.Metadata.Get(key)
	if rawID == "" {
		return "", fmt.Errorf("%s not found in metadata", key)
	}

	return rawID, nil
}

// ParseEntityEvent parses a message.Message and returns an EntityInfoWrapper
//
//nolint:gocyclo // This will be simplified once we rely solely on the entity ID key
func ParseEntityEvent(msg *message.Message) (*EntityInfoWrapper, error) {
	out := &EntityInfoWrapper{
		OwnershipData: make(map[string]string),
	}

	if err := out.withProjectIDFromMessage(msg); err != nil {
		return nil, err
	}

	if err := out.withProviderIDFromMessage(msg); err != nil {
		return nil, err
	}

	if err := out.withEntityInstanceIDFromMessage(msg); err != nil {
		// We don't fail, but instead log the error and continue
		// We'll fall back to the other entity ID keys.
		zerolog.Ctx(msg.Context()).Debug().
			Str("message_id", msg.UUID).
			Msg("message does not contain entity ID")
	}

	// We don't always have repo ID (e.g. for artifacts)

	typ := msg.Metadata.Get(EntityTypeEventKey)
	strtyp := minderv1.EntityFromString(typ)

	//nolint:exhaustive // We have a default case
	switch strtyp {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		out.AsRepository()
		if out.EntityID == uuid.Nil {
			if err := out.withRepositoryIDFromMessage(msg); err != nil {
				return nil, err
			}
		}
	case minderv1.Entity_ENTITY_ARTIFACTS:
		out.AsArtifact()
		if out.EntityID == uuid.Nil {
			if err := out.withArtifactIDFromMessage(msg); err != nil {
				return nil, err
			}
			//nolint:gosec // The repo is not always present
			out.withRepositoryIDFromMessage(msg)
		}
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		out.AsPullRequest()
		if out.EntityID == uuid.Nil {
			if err := out.withPullRequestIDFromMessage(msg); err != nil {
				return nil, err
			}
			if err := out.withRepositoryIDFromMessage(msg); err != nil {
				return nil, err
			}
		}
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		return nil, fmt.Errorf("entity type unspecified")
	default:
		// We can't fall back in this case.
		if out.EntityID == uuid.Nil {
			return nil, fmt.Errorf("entity ID not found")
		}

		// Any other entity type
		out.AsEntityInstance(strtyp)
		if err := out.withEntityInstanceIDFromMessage(msg); err != nil {
			return nil, err
		}
	}

	if err := out.unmarshalEntity(msg); err != nil {
		return nil, fmt.Errorf("error unmarshalling payload: %w", err)
	}

	return out, nil
}
