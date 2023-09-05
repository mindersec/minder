// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/entities"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// entityInfoWrapper is a helper struct to gather information
// about entities from events.
// It assumes that the message.Message contains a payload
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
type entityInfoWrapper struct {
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
)

const (
	// EntityTypeEventKey is the key for the entity type
	EntityTypeEventKey = "entity_type"
	// GroupIDEventKey is the key for the group ID
	GroupIDEventKey = "group_id"
	// RepositoryIDEventKey is the key for the repository ID
	RepositoryIDEventKey = "repository_id"
	// ArtifactIDEventKey is the key for the artifact ID
	ArtifactIDEventKey = "artifact_id"
)

func parseEntityEvent(msg *message.Message) (*entityInfoWrapper, error) {
	out := &entityInfoWrapper{
		OwnershipData: make(map[string]int32),
	}

	if err := out.withGroupIDFromMessage(msg); err != nil {
		return nil, err
	}

	// We always have the repository ID.
	if err := out.withRepositoryIDFromMessage(msg); err != nil {
		return nil, err
	}

	typ := msg.Metadata.Get(EntityTypeEventKey)
	switch typ {
	case RepositoryEventEntityType:
		out.asRepository()
	case VersionedArtifactEventEntityType:
		out.asVersionedArtifact()
		if err := out.withArtifactIDFromMessage(msg); err != nil {
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

func (eiw *entityInfoWrapper) asRepository() {
	eiw.Type = pb.Entity_ENTITY_REPOSITORIES
	eiw.Entity = &pb.RepositoryResult{}
}

func (eiw *entityInfoWrapper) asVersionedArtifact() {
	eiw.Type = pb.Entity_ENTITY_ARTIFACTS
	eiw.Entity = &pb.VersionedArtifact{}
}

func (eiw *entityInfoWrapper) withGroupIDFromMessage(msg *message.Message) error {
	id, err := getIDFromMessage(msg, GroupIDEventKey)
	if err != nil {
		return fmt.Errorf("error parsing %s: %w", GroupIDEventKey, err)
	}

	eiw.GroupID = id
	return nil
}

func (eiw *entityInfoWrapper) withRepositoryIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, RepositoryIDEventKey)
}

func (eiw *entityInfoWrapper) withArtifactIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, ArtifactIDEventKey)
}

func (eiw *entityInfoWrapper) withIDFromMessage(msg *message.Message, key string) error {
	id, err := getIDFromMessage(msg, key)
	if err != nil {
		return fmt.Errorf("error parsing %s: %w", key, err)
	}

	eiw.OwnershipData[key] = id
	return nil
}

func (eiw *entityInfoWrapper) unmarshalEntity(msg *message.Message) error {
	return protojson.Unmarshal(msg.Payload, eiw.Entity)
}

func (eiw *entityInfoWrapper) evalStatusParams(
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

	return params
}

func getIDFromMessage(msg *message.Message, key string) (int32, error) {
	rawID := msg.Metadata.Get(key)
	if rawID == "" {
		return 0, fmt.Errorf("%s not found in metadata", key)
	}

	return util.Int32FromString(rawID)
}
