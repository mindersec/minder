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
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func Test_parseEntityEvent(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()
	repoID := uuid.NewString()
	prID := uuid.NewString()
	artifactID := uuid.NewString()

	type args struct {
		ent        protoreflect.ProtoMessage
		entType    string
		projectID  uuid.UUID
		providerID uuid.UUID
		entityID   string
		ownership  map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    *EntityInfoWrapper
		wantErr bool
	}{
		{
			name: "legacy repository event",
			args: args{
				ent: &pb.Repository{
					Name:   "test",
					RepoId: 123,
				},
				entType:    pb.Entity_ENTITY_REPOSITORIES.ToString(),
				projectID:  projectID,
				providerID: providerID,
				ownership:  map[string]string{repositoryIDEventKey: repoID},
			},
			want: &EntityInfoWrapper{
				ProjectID: projectID,
				Entity: &pb.Repository{
					Name:   "test",
					RepoId: 123,
				},
				ProviderID: providerID,
				Type:       pb.Entity_ENTITY_REPOSITORIES,
				OwnershipData: map[string]string{
					repositoryIDEventKey: repoID,
				},
			},
		},
		{
			name: "legacy versioned artifact event",
			args: args{
				ent: &pb.Artifact{
					ArtifactPk: artifactID,
					Versions: []*pb.ArtifactVersion{
						{
							VersionId: 789,
						},
					},
				},
				entType:    pb.Entity_ENTITY_ARTIFACTS.ToString(),
				projectID:  projectID,
				providerID: providerID,
				ownership: map[string]string{
					repositoryIDEventKey: repoID,
					artifactIDEventKey:   artifactID,
				},
			},
			want: &EntityInfoWrapper{
				ProjectID: projectID,
				Entity: &pb.Artifact{
					ArtifactPk: artifactID,
					Versions: []*pb.ArtifactVersion{
						{
							VersionId: 789,
						},
					},
				},
				ProviderID: providerID,
				Type:       pb.Entity_ENTITY_ARTIFACTS,
				OwnershipData: map[string]string{
					repositoryIDEventKey: repoID,
					artifactIDEventKey:   artifactID,
				},
			},
		},
		{
			name: "legacy pull_request event",
			args: args{
				ent: &pb.PullRequest{
					Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/3",
					CommitSha: "bd9958a63c9b95ccc2bc0cf1eef65a87529aed16",
					Number:    3,
					RepoOwner: "jakubtestorg",
					RepoName:  "bad-npm",
				},
				entType:    pb.Entity_ENTITY_PULL_REQUESTS.ToString(),
				projectID:  projectID,
				providerID: providerID,
				ownership: map[string]string{
					pullRequestIDEventKey: prID,
					repositoryIDEventKey:  repoID,
				},
			},
			want: &EntityInfoWrapper{
				ProjectID: projectID,
				Entity: &pb.PullRequest{
					Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/3",
					CommitSha: "bd9958a63c9b95ccc2bc0cf1eef65a87529aed16",
					Number:    3,
					RepoOwner: "jakubtestorg",
					RepoName:  "bad-npm",
				},
				ProviderID: providerID,
				Type:       pb.Entity_ENTITY_PULL_REQUESTS,
				OwnershipData: map[string]string{
					pullRequestIDEventKey: prID,
					repositoryIDEventKey:  repoID,
				},
			},
		},
		{
			name: "repository event with entity ID",
			args: args{
				ent: &pb.Repository{
					Name:   "test",
					RepoId: 123,
				},
				entType:    pb.Entity_ENTITY_REPOSITORIES.ToString(),
				projectID:  projectID,
				providerID: providerID,
				entityID:   repoID,
			},
			want: &EntityInfoWrapper{
				ProjectID: projectID,
				Entity: &pb.Repository{
					Name:   "test",
					RepoId: 123,
				},
				ProviderID:    providerID,
				Type:          pb.Entity_ENTITY_REPOSITORIES,
				EntityID:      uuid.MustParse(repoID),
				OwnershipData: map[string]string{},
			},
		},
		{
			name: "artifact event with entity ID",
			args: args{
				ent: &pb.Artifact{
					ArtifactPk: artifactID,
					Versions: []*pb.ArtifactVersion{
						{
							VersionId: 789,
						},
					},
				},
				entType:    pb.Entity_ENTITY_ARTIFACTS.ToString(),
				projectID:  projectID,
				providerID: providerID,
				entityID:   artifactID,
			},
			want: &EntityInfoWrapper{
				ProjectID: projectID,
				Entity: &pb.Artifact{
					ArtifactPk: artifactID,
					Versions: []*pb.ArtifactVersion{
						{
							VersionId: 789,
						},
					},
				},
				ProviderID:    providerID,
				Type:          pb.Entity_ENTITY_ARTIFACTS,
				EntityID:      uuid.MustParse(artifactID),
				OwnershipData: map[string]string{},
			},
		},
		{
			name: "pull_request event with entity ID",
			args: args{
				ent: &pb.PullRequest{
					Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/3",
					CommitSha: "bd9958a63c9b95ccc2bc0cf1eef65a87529aed16",
					Number:    3,
					RepoOwner: "jakubtestorg",
					RepoName:  "bad-npm",
				},
				entType:    pb.Entity_ENTITY_PULL_REQUESTS.ToString(),
				projectID:  projectID,
				providerID: providerID,
				entityID:   prID,
			},
			want: &EntityInfoWrapper{
				ProjectID: projectID,
				Entity: &pb.PullRequest{
					Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/3",
					CommitSha: "bd9958a63c9b95ccc2bc0cf1eef65a87529aed16",
					Number:    3,
					RepoOwner: "jakubtestorg",
					RepoName:  "bad-npm",
				},
				ProviderID:    providerID,
				Type:          pb.Entity_ENTITY_PULL_REQUESTS,
				EntityID:      uuid.MustParse(prID),
				OwnershipData: map[string]string{},
			},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			marshalledEntity, err := protojson.Marshal(tt.args.ent)
			require.NoError(t, err, "unexpected error")

			msg := message.NewMessage("", marshalledEntity)
			msg.Metadata.Set(ProjectIDEventKey, tt.args.projectID.String())
			msg.Metadata.Set(EntityTypeEventKey, tt.args.entType)
			msg.Metadata.Set(ProviderIDEventKey, tt.args.providerID.String())

			if len(tt.args.entityID) > 0 {
				msg.Metadata.Set(EntityIDEventKey, tt.args.entityID)
			}

			if len(tt.args.ownership) > 0 {
				msg.Metadata.Set(repositoryIDEventKey, tt.args.ownership["repository_id"])
				if tt.args.entType == pb.Entity_ENTITY_ARTIFACTS.ToString() {
					msg.Metadata.Set(artifactIDEventKey, tt.args.ownership["artifact_id"])
				} else if tt.args.entType == pb.Entity_ENTITY_PULL_REQUESTS.ToString() {
					msg.Metadata.Set(pullRequestIDEventKey, tt.args.ownership["pull_request_id"])
				}
			}

			got, err := ParseEntityEvent(msg)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, got, "expected nil entity info")
				return
			}

			require.NoError(t, err, "unexpected error")
			require.NotNil(t, got, "expected non-nil entity info")

			//NOTE: Not checking the entity right now because it's a pain to compare
			assert.Equal(t, tt.want.ProjectID, got.ProjectID, "project id mismatch")
			assert.Equal(t, tt.want.Type, got.Type, "entity type mismatch")
			assert.Equal(t, tt.want.OwnershipData, got.OwnershipData, "ownership data mismatch")
			assert.Equal(t, tt.want.ProviderID, got.ProviderID, "provider ID mismatch")
		})
	}
}

func TestEntityInfoWrapper_ToMessage(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	repoID := uuid.New()
	providerID := uuid.New()
	artifactID := uuid.New()
	pullRequestID := uuid.New()

	tests := []struct {
		name     string
		eiw      *EntityInfoWrapper
		expected map[string]string
	}{
		{
			name: "repository event",
			eiw: NewEntityInfoWrapper().
				WithProviderID(providerID).
				WithProjectID(projectID).
				WithRepository(&pb.Repository{
					Owner:  "test",
					RepoId: 123,
				}).WithID(repoID),
			expected: map[string]string{
				ProviderIDEventKey: providerID.String(),
				EntityTypeEventKey: pb.Entity_ENTITY_REPOSITORIES.ToString(),
				ProjectIDEventKey:  projectID.String(),
				EntityIDEventKey:   repoID.String(),
			},
		},
		{
			name: "repository using generic protobuf message and id",
			eiw: NewEntityInfoWrapper().
				WithProviderID(providerID).
				WithProjectID(projectID).
				WithProtoMessage(pb.Entity_ENTITY_REPOSITORIES, &pb.Repository{
					Owner:  "test",
					RepoId: 123,
				}).
				WithID(repoID),
			expected: map[string]string{
				ProviderIDEventKey: providerID.String(),
				EntityTypeEventKey: pb.Entity_ENTITY_REPOSITORIES.ToString(),
				ProjectIDEventKey:  projectID.String(),
				EntityIDEventKey:   repoID.String(),
			},
		},
		{
			name: "artifact event",
			eiw: NewEntityInfoWrapper().
				WithProviderID(providerID).
				WithProjectID(projectID).
				WithArtifact(&pb.Artifact{
					ArtifactPk: artifactID.String(),
					Versions: []*pb.ArtifactVersion{
						{
							VersionId: 101112,
						},
					},
				}).
				WithID(artifactID),
			expected: map[string]string{
				ProviderIDEventKey: providerID.String(),
				EntityTypeEventKey: pb.Entity_ENTITY_ARTIFACTS.ToString(),
				ProjectIDEventKey:  projectID.String(),
				EntityIDEventKey:   artifactID.String(),
			},
		},
		{
			name: "pull request using WithID and WithProtoMessage",
			eiw: NewEntityInfoWrapper().
				WithProviderID(providerID).
				WithProjectID(projectID).
				WithProtoMessage(pb.Entity_ENTITY_PULL_REQUESTS, &pb.PullRequest{
					Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/3",
					CommitSha: "bd9958a63c9b95ccc2bc0cf1eef65a87529aed16",
					Number:    3,
					RepoOwner: "jakubtestorg",
					RepoName:  "bad-npm",
				}).
				WithID(pullRequestID),
			expected: map[string]string{
				ProviderIDEventKey: providerID.String(),
				EntityTypeEventKey: pb.Entity_ENTITY_PULL_REQUESTS.ToString(),
				ProjectIDEventKey:  projectID.String(),
				EntityIDEventKey:   pullRequestID.String(),
			},
		},
		{
			name: "pull request using WithPullRequest",
			eiw: NewEntityInfoWrapper().
				WithProviderID(providerID).
				WithProjectID(projectID).
				WithPullRequest(&pb.PullRequest{
					Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/3",
					CommitSha: "bd9958a63c9b95ccc2bc0cf1eef65a87529aed16",
					Number:    3,
					RepoOwner: "jakubtestorg",
					RepoName:  "bad-npm",
				}).
				WithID(pullRequestID),
			expected: map[string]string{
				ProviderIDEventKey: providerID.String(),
				EntityTypeEventKey: pb.Entity_ENTITY_PULL_REQUESTS.ToString(),
				ProjectIDEventKey:  projectID.String(),
				EntityIDEventKey:   pullRequestID.String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg, err := tt.eiw.BuildMessage()
			require.NoError(t, err, "unexpected error")

			for key, expectedValue := range tt.expected {
				assert.Equal(t, expectedValue, msg.Metadata.Get(key), key+" mismatch")
			}
		})
	}
}

func TestEntityInfoWrapper_FailsWithoutProjectID(t *testing.T) {
	t.Parallel()

	eiw := NewEntityInfoWrapper().
		WithProviderID(uuid.New()).
		WithRepository(&pb.Repository{
			Owner:  "test",
			RepoId: 123,
		}).WithID(uuid.New())

	msg, err := eiw.BuildMessage()
	t.Logf("OZZ: %+v", msg)
	require.Error(t, err, "expected error")
}

func TestEntityInfoWrapper_FailsWithoutProvider(t *testing.T) {
	t.Parallel()

	eiw := NewEntityInfoWrapper().
		WithProjectID(uuid.New()).
		WithRepository(&pb.Repository{
			Owner:  "test",
			RepoId: 123,
		}).WithID(uuid.New())

	_, err := eiw.BuildMessage()
	require.Error(t, err, "expected error")
}

func TestEntityInfoWrapper_FailsWithoutRepository(t *testing.T) {
	t.Parallel()

	eiw := NewEntityInfoWrapper().
		WithProviderID(uuid.New()).
		WithProjectID(uuid.New()).
		WithID(uuid.New())

	_, err := eiw.BuildMessage()
	require.Error(t, err, "expected error")
}

func TestEntityInfoWrapper_FailsWithInvalidEntity(t *testing.T) {
	t.Parallel()

	eiw := NewEntityInfoWrapper().
		WithProviderID(uuid.New()).
		WithProjectID(uuid.New())

	eiw.Entity = &pb.UserRecord{}

	_, err := eiw.BuildMessage()
	require.Error(t, err, "expected error")
}
