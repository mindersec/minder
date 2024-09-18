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
// - ProjectIDEventKey - project_id
// - RepositoryIDEventKey - repository_id
// - ArtifactIDEventKey - artifact_id (only for versioned artifacts)
//
// Entity type is used to determine the type of the protobuf message
// and the entity type in the database. It may be one of the following:
//
// - RepositoryEventEntityType - repository
// - VersionedArtifactEventEntityType - versioned_artifact
type EntityInfoWrapper struct {
	ProviderID    uuid.UUID
	ProjectID     uuid.UUID
	Entity        protoreflect.ProtoMessage
	Type          minderv1.Entity
	OwnershipData map[string]string
	ExecutionID   *uuid.UUID
	ActionEvent   string
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
	// ProviderIDEventKey is the key for the provider ID
	ProviderIDEventKey = "provider_id"
	// ProjectIDEventKey is the key for the project ID
	ProjectIDEventKey = "project_id"
	// RepositoryIDEventKey is the key for the repository ID
	RepositoryIDEventKey = "repository_id"
	// ArtifactIDEventKey is the key for the artifact ID
	ArtifactIDEventKey = "artifact_id"
	// PullRequestIDEventKey is the key for the pull request ID
	PullRequestIDEventKey = "pull_request_id"
	// ReleaseIDEventKey is the key for the pull request ID
	ReleaseIDEventKey = "release_id"
	// PipelineRunIDEventKey is the key for a pipeline run
	PipelineRunIDEventKey = "pipeline_run_id"
	// TaskRunIDEventKey is the key for a task run
	TaskRunIDEventKey = "task_run_id"
	// BuildIDEventKey is the key for a build
	BuildIDEventKey = "build_run_id"
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

// WithRelease sets a Release as the entity of the wrapper
func (eiw *EntityInfoWrapper) WithRelease(r *minderv1.Release) *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_RELEASE
	eiw.Entity = r

	return eiw
}

// WithPipelineRun sets a PipelineRun as the entity of the wrapper
func (eiw *EntityInfoWrapper) WithPipelineRun(plr *minderv1.PipelineRun) *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_PIPELINE_RUN
	eiw.Entity = plr

	return eiw
}

// WithTaskRun sets a TaskRun as the entity of the wrapper
func (eiw *EntityInfoWrapper) WithTaskRun(tr *minderv1.TaskRun) *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_TASK_RUN
	eiw.Entity = tr

	return eiw
}

// WithBuild sets a Build as the entity of the wrapper
func (eiw *EntityInfoWrapper) WithBuild(tr *minderv1.Build) *EntityInfoWrapper {
	eiw.Type = minderv1.Entity_ENTITY_TASK_RUN
	eiw.Entity = tr

	return eiw
}

// WithProjectID sets the project ID
func (eiw *EntityInfoWrapper) WithProjectID(id uuid.UUID) *EntityInfoWrapper {
	eiw.ProjectID = id

	return eiw
}

// WithRepositoryID sets the repository ID
func (eiw *EntityInfoWrapper) WithRepositoryID(id uuid.UUID) *EntityInfoWrapper {
	eiw.withID(RepositoryIDEventKey, id.String())

	return eiw
}

// WithArtifactID sets the artifact ID
func (eiw *EntityInfoWrapper) WithArtifactID(id uuid.UUID) *EntityInfoWrapper {
	eiw.withID(ArtifactIDEventKey, id.String())

	return eiw
}

// WithPullRequestID sets the pull request ID
func (eiw *EntityInfoWrapper) WithPullRequestID(id uuid.UUID) *EntityInfoWrapper {
	eiw.withID(PullRequestIDEventKey, id.String())

	return eiw
}

// WithReleaseID sets the release ID
func (eiw *EntityInfoWrapper) WithReleaseID(id uuid.UUID) *EntityInfoWrapper {
	eiw.withID(ReleaseIDEventKey, id.String())

	return eiw
}

// WithPipelineRunID sets the pipeline run ID
func (eiw *EntityInfoWrapper) WithPipelineRunID(id uuid.UUID) *EntityInfoWrapper {
	eiw.withID(PipelineRunIDEventKey, id.String())

	return eiw
}

// WithTaskRunID sets the pipeline run ID
func (eiw *EntityInfoWrapper) WithTaskRunID(id uuid.UUID) *EntityInfoWrapper {
	eiw.withID(TaskRunIDEventKey, id.String())

	return eiw
}

// WithBuildID sets the pipeline run ID
func (eiw *EntityInfoWrapper) WithBuildID(id uuid.UUID) *EntityInfoWrapper {
	eiw.withID(BuildIDEventKey, id.String())

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
	typ, err := pbEntityTypeToString(eiw.Type)
	if err != nil {
		return err
	}

	if eiw.ProjectID == uuid.Nil {
		return fmt.Errorf("project ID is required")
	}

	if eiw.ProviderID == uuid.Nil {
		return fmt.Errorf("provider ID is required")
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
	msg.Payload, err = protojson.Marshal(eiw.Entity)
	if err != nil {
		return fmt.Errorf("error marshalling repository: %w", err)
	}

	return nil
}

// GetEntityDBIDs returns the repository, artifact and pull request IDs
// from the ownership data
func (eiw *EntityInfoWrapper) GetEntityDBIDs() (repoID uuid.NullUUID, artifactID uuid.NullUUID, pullRequestID uuid.NullUUID) {
	strRepoID, ok := eiw.OwnershipData[RepositoryIDEventKey]
	if ok {
		repoID = uuid.NullUUID{
			UUID:  uuid.MustParse(strRepoID),
			Valid: true,
		}
	}

	strArtifactID, ok := eiw.OwnershipData[ArtifactIDEventKey]
	if ok {
		artifactID = uuid.NullUUID{
			UUID:  uuid.MustParse(strArtifactID),
			Valid: true,
		}
	}

	strPullRequestID, ok := eiw.OwnershipData[PullRequestIDEventKey]
	if ok {
		pullRequestID = uuid.NullUUID{
			UUID:  uuid.MustParse(strPullRequestID),
			Valid: true,
		}
	}

	return repoID, artifactID, pullRequestID
}

// GetID returns the entity ID.
func (eiw *EntityInfoWrapper) GetID() (uuid.UUID, error) {
	if eiw == nil {
		return uuid.Nil, fmt.Errorf("no entity info wrapper")
	}

	id, ok := eiw.getIDForEntityType(eiw.Type)
	if ok {
		return id, nil
	}

	return uuid.Nil, fmt.Errorf("no entity ID found")
}

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
	return eiw.withIDFromMessage(msg, RepositoryIDEventKey)
}

func (eiw *EntityInfoWrapper) withArtifactIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, ArtifactIDEventKey)
}

func (eiw *EntityInfoWrapper) withPullRequestIDFromMessage(msg *message.Message) error {
	return eiw.withIDFromMessage(msg, PullRequestIDEventKey)
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

func pbEntityTypeToString(t minderv1.Entity) (string, error) {
	switch t {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return RepositoryEventEntityType, nil
	case minderv1.Entity_ENTITY_ARTIFACTS:
		return VersionedArtifactEventEntityType, nil
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return PullRequestEventEntityType, nil
	case minderv1.Entity_ENTITY_RELEASE:
		return "", fmt.Errorf("releases not yet supported")
	case minderv1.Entity_ENTITY_PIPELINE_RUN:
		return "", fmt.Errorf("pipeline runs not yet supported")
	case minderv1.Entity_ENTITY_TASK_RUN:
		return "", fmt.Errorf("task runs not yet supported")
	case minderv1.Entity_ENTITY_BUILD:
		return "", fmt.Errorf("builds not yet supported")
	case minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return "", fmt.Errorf("build environments not yet supported")
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		return "", fmt.Errorf("entity type unspecified")
	default:
		return "", fmt.Errorf("unknown entity type: %s", t.String())
	}
}

func getEntityMetadataKey(t minderv1.Entity) (string, error) {
	switch t {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return RepositoryIDEventKey, nil
	case minderv1.Entity_ENTITY_ARTIFACTS:
		return ArtifactIDEventKey, nil
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return PullRequestIDEventKey, nil
	case minderv1.Entity_ENTITY_RELEASE:
		return ReleaseIDEventKey, nil
	case minderv1.Entity_ENTITY_PIPELINE_RUN:
		return PipelineRunIDEventKey, nil
	case minderv1.Entity_ENTITY_TASK_RUN:
		return TaskRunIDEventKey, nil
	case minderv1.Entity_ENTITY_BUILD:
		return BuildIDEventKey, nil
	case minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return "", fmt.Errorf("build environments not yet supported")
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		return "", fmt.Errorf("entity type unspecified")
	default:
		return "", fmt.Errorf("unknown entity type: %s", t.String())
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

	// We don't always have repo ID (e.g. for artifacts)

	typ := msg.Metadata.Get(EntityTypeEventKey)
	switch typ {
	case RepositoryEventEntityType:
		out.AsRepository()
		if err := out.withRepositoryIDFromMessage(msg); err != nil {
			return nil, err
		}
	case VersionedArtifactEventEntityType:
		out.AsArtifact()
		if err := out.withArtifactIDFromMessage(msg); err != nil {
			return nil, err
		}
		//nolint:gosec // The repo is not always present
		out.withRepositoryIDFromMessage(msg)
	case PullRequestEventEntityType:
		out.AsPullRequest()
		if err := out.withPullRequestIDFromMessage(msg); err != nil {
			return nil, err
		}
		if err := out.withRepositoryIDFromMessage(msg); err != nil {
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

// WithID sets the ID for an entity type
func (eiw *EntityInfoWrapper) WithID(entType minderv1.Entity, id uuid.UUID) *EntityInfoWrapper {
	key, err := getEntityMetadataKey(entType)
	if err != nil {
		return nil
	}

	eiw.withID(key, id.String())
	return eiw
}
