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

package engine

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/internal/entities"
	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
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
// - GroupIDEventKey - group_id
// - RepositoryIDEventKey - repository_id
// - ArtifactIDEventKey - artifact_id (only for versioned artifacts)
//
// Entity type is used to determine the type of the protobuf message
// and the entity type in the database. It may be one of the following:
//
// - RepositoryEventEntityType - repository
// - VersionedArtifactEventEntityType - versioned_artifact
type EntityInfoWrapper struct {
	Provider      string
	GroupID       int32
	Entity        protoreflect.ProtoMessage
	Type          pb.Entity
	OwnershipData map[string]int32
}

const (
	// RepositoryEventEntityType is the entity type for repositories
	RepositoryEventEntityType = "repository"
	// VersionedArtifactEventEntityType is the entity type for versioned artifacts
	VersionedArtifactEventEntityType = "versioned_artifact"
	// PullRequestEventEntityType is the entity type for pull requests
	PullRequestEventEntityType = "pull_request"
)

const (
	// EntityTypeEventKey is the key for the entity type
	EntityTypeEventKey = "entity_type"
	// ProviderEventKey is the key for the provider
	ProviderEventKey = "provider"
	// GroupIDEventKey is the key for the group ID
	GroupIDEventKey = "group_id"
	// RepositoryIDEventKey is the key for the repository ID
	RepositoryIDEventKey = "repository_id"
	// ArtifactIDEventKey is the key for the artifact ID
	ArtifactIDEventKey = "artifact_id"
	// PullRequestIDEventKey is the key for the pull request ID
	PullRequestIDEventKey = "pull_request_id"
)

// NewEntityInfoWrapper creates a new EntityInfoWrapper
func NewEntityInfoWrapper() *EntityInfoWrapper {
	return &EntityInfoWrapper{
		OwnershipData: make(map[string]int32),
	}
}

// WithProvider sets the provider
func (eiw *EntityInfoWrapper) WithProvider(provider string) *EntityInfoWrapper {
	eiw.Provider = provider

	return eiw
}

// WithVersionedArtifact sets the entity to a versioned artifact
func (eiw *EntityInfoWrapper) WithVersionedArtifact(va *pb.VersionedArtifact) *EntityInfoWrapper {
	eiw.Type = pb.Entity_ENTITY_ARTIFACTS
	eiw.Entity = va

	return eiw
}

// WithRepository sets the entity to a repository
func (eiw *EntityInfoWrapper) WithRepository(r *pb.RepositoryResult) *EntityInfoWrapper {
	eiw.Type = pb.Entity_ENTITY_REPOSITORIES
	eiw.Entity = r

	return eiw
}

// WithPullRequest sets the entity to a repository
func (eiw *EntityInfoWrapper) WithPullRequest(p *pb.PullRequest) *EntityInfoWrapper {
	eiw.Type = pb.Entity_ENTITY_PULL_REQUESTS
	eiw.Entity = p

	return eiw
}

// WithGroupID sets the group ID
func (eiw *EntityInfoWrapper) WithGroupID(id int32) *EntityInfoWrapper {
	eiw.GroupID = id

	return eiw
}

// WithRepositoryID sets the repository ID
func (eiw *EntityInfoWrapper) WithRepositoryID(id int32) *EntityInfoWrapper {
	eiw.withID(RepositoryIDEventKey, id)

	return eiw
}

// WithArtifactID sets the artifact ID
func (eiw *EntityInfoWrapper) WithArtifactID(id int32) *EntityInfoWrapper {
	eiw.withID(ArtifactIDEventKey, id)

	return eiw
}

// WithPullRequestID sets the pull request ID
func (eiw *EntityInfoWrapper) WithPullRequestID(id int32) *EntityInfoWrapper {
	eiw.withID(PullRequestIDEventKey, id)

	return eiw
}

// AsRepository sets the entity type to a repository
func (eiw *EntityInfoWrapper) AsRepository() *EntityInfoWrapper {
	eiw.Type = pb.Entity_ENTITY_REPOSITORIES
	eiw.Entity = &pb.RepositoryResult{}

	return eiw
}

// AsVersionedArtifact sets the entity type to a versioned artifact
func (eiw *EntityInfoWrapper) AsVersionedArtifact() *EntityInfoWrapper {
	eiw.Type = pb.Entity_ENTITY_ARTIFACTS
	eiw.Entity = &pb.VersionedArtifact{}

	return eiw
}

// AsPullRequest sets the entity type to a pull request
func (eiw *EntityInfoWrapper) AsPullRequest() {
	eiw.Type = pb.Entity_ENTITY_PULL_REQUESTS
	eiw.Entity = &pb.PullRequest{}
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
func (eiw *EntityInfoWrapper) Publish(evt *events.Eventer) error {
	msg, err := eiw.BuildMessage()
	if err != nil {
		return err
	}

	if err := evt.Publish(InternalEntityEventTopic, msg); err != nil {
		return fmt.Errorf("error publishing entity event: %w", err)
	}

	return nil
}

// ToMessage sets the information to a message.Message
func (eiw *EntityInfoWrapper) ToMessage(msg *message.Message) error {
	typ, err := pbEntityTypeToString(eiw.Type)
	if err != nil {
		return err
	}

	if eiw.GroupID == 0 {
		return fmt.Errorf("group ID is required")
	}

	if eiw.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	msg.Metadata.Set(ProviderEventKey, eiw.Provider)
	msg.Metadata.Set(EntityTypeEventKey, typ)
	msg.Metadata.Set(GroupIDEventKey, fmt.Sprintf("%d", eiw.GroupID))
	for k, v := range eiw.OwnershipData {
		if v == 0 {
			return fmt.Errorf("%s is required", k)
		}
		msg.Metadata.Set(k, fmt.Sprintf("%d", v))
	}
	msg.Payload, err = protojson.Marshal(eiw.Entity)
	if err != nil {
		return fmt.Errorf("error marshalling repository: %w", err)
	}

	return nil
}

func (eiw *EntityInfoWrapper) withGroupIDFromMessage(msg *message.Message) error {
	id, err := getIDFromMessage(msg, GroupIDEventKey)
	if err != nil {
		return fmt.Errorf("error parsing %s: %w", GroupIDEventKey, err)
	}

	eiw.GroupID = id
	return nil
}

func (eiw *EntityInfoWrapper) withProviderFromMessage(msg *message.Message) error {
	provider := msg.Metadata.Get(ProviderEventKey)
	if provider == "" {
		return fmt.Errorf("%s not found in metadata", ProviderEventKey)
	}

	eiw.Provider = provider
	return nil
}

func (eiw *EntityInfoWrapper) withRepositoryIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, RepositoryIDEventKey)
}

func (eiw *EntityInfoWrapper) withArtifactIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, ArtifactIDEventKey)
}

func (eiw *EntityInfoWrapper) withPullRequestIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, PullRequestIDEventKey)
}

func (eiw *EntityInfoWrapper) withIDFromMessage(msg *message.Message, key string) error {
	id, err := getIDFromMessage(msg, key)
	if err != nil {
		return fmt.Errorf("error parsing %s: %w", key, err)
	}

	eiw.OwnershipData[key] = id
	return nil
}

func (eiw *EntityInfoWrapper) withID(key string, id int32) {
	eiw.OwnershipData[key] = id
}

func (eiw *EntityInfoWrapper) unmarshalEntity(msg *message.Message) error {
	return protojson.Unmarshal(msg.Payload, eiw.Entity)
}

func (eiw *EntityInfoWrapper) evalStatusParams(
	policyID int32,
	ruleTypeID int32,
	evalErr error,
) *createOrUpdateEvalStatusParams {
	params := &createOrUpdateEvalStatusParams{
		policyID:       policyID,
		repoID:         eiw.OwnershipData[RepositoryIDEventKey],
		ruleTypeEntity: entities.EntityTypeToDB(eiw.Type),
		ruleTypeID:     ruleTypeID,
		evalErr:        evalErr,
	}

	artifactID, ok := eiw.OwnershipData[ArtifactIDEventKey]
	if ok {
		params.artifactID = artifactID
	}

	pullRequestNumber, ok := eiw.OwnershipData[PullRequestIDEventKey]
	if ok {
		// todo: plug into DB
		fmt.Println("pullRequestNumber", pullRequestNumber)
	}

	return params
}

func pbEntityTypeToString(t pb.Entity) (string, error) {
	switch t {
	case pb.Entity_ENTITY_REPOSITORIES:
		return RepositoryEventEntityType, nil
	case pb.Entity_ENTITY_ARTIFACTS:
		return VersionedArtifactEventEntityType, nil
	case pb.Entity_ENTITY_PULL_REQUESTS:
		return PullRequestEventEntityType, nil
	case pb.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return "", fmt.Errorf("build environments not yet supported")
	case pb.Entity_ENTITY_UNSPECIFIED:
		return "", fmt.Errorf("entity type unspecified")
	default:
		return "", fmt.Errorf("unknown entity type: %s", t.String())
	}
}

func getIDFromMessage(msg *message.Message, key string) (int32, error) {
	rawID := msg.Metadata.Get(key)
	if rawID == "" {
		return 0, fmt.Errorf("%s not found in metadata", key)
	}

	return util.Int32FromString(rawID)
}

func parseEntityEvent(msg *message.Message) (*EntityInfoWrapper, error) {
	out := &EntityInfoWrapper{
		OwnershipData: make(map[string]int32),
	}

	if err := out.withGroupIDFromMessage(msg); err != nil {
		return nil, err
	}

	if err := out.withProviderFromMessage(msg); err != nil {
		return nil, err
	}

	// We always have the repository ID.
	if err := out.withRepositoryIDFromMessage(msg); err != nil {
		return nil, err
	}

	typ := msg.Metadata.Get(EntityTypeEventKey)
	switch typ {
	case RepositoryEventEntityType:
		out.AsRepository()
	case VersionedArtifactEventEntityType:
		out.AsVersionedArtifact()
		if err := out.withArtifactIDFromMessage(msg); err != nil {
			return nil, err
		}
	case PullRequestEventEntityType:
		out.AsPullRequest()
		if err := out.withPullRequestIDFromMessage(msg); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown entity type: %s", typ)
	}

	if err := out.unmarshalEntity(msg); err != nil {
		return nil, fmt.Errorf("error unmarshalling payload: %w", err)
	}

	return out, nil
}
