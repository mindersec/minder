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
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func Test_parseEntityEvent(t *testing.T) {
	t.Parallel()

	type args struct {
		ent       protoreflect.ProtoMessage
		entType   string
		groupID   int32
		ownership map[string]int32
	}
	tests := []struct {
		name    string
		args    args
		want    *entityInfoWrapper
		wantErr bool
	}{
		{
			name: "repository event",
			args: args{
				ent: &pb.RepositoryResult{
					Repository: "test",
					RepoId:     123,
				},
				entType:   RepositoryEventEntityType,
				groupID:   1,
				ownership: map[string]int32{RepositoryIDEventKey: 123},
			},
			want: &entityInfoWrapper{
				GroupID: 1,
				Entity: &pb.RepositoryResult{
					Repository: "test",
					RepoId:     123,
				},
				Type:          pb.Entity_ENTITY_REPOSITORIES,
				OwnershipData: map[string]int32{RepositoryIDEventKey: 123},
			},
		},
		{
			name: "versioned artifact event",
			args: args{
				ent: &pb.VersionedArtifact{
					Artifact: &pb.Artifact{
						ArtifactPk: 123,
					},
					Version: &pb.ArtifactVersion{
						VersionId: 789,
					},
				},
				entType:   VersionedArtifactEventEntityType,
				groupID:   1,
				ownership: map[string]int32{RepositoryIDEventKey: 456, ArtifactIDEventKey: 123},
			},
			want: &entityInfoWrapper{
				GroupID: 1,
				Entity: &pb.VersionedArtifact{
					Artifact: &pb.Artifact{
						ArtifactPk: 123,
					},
					Version: &pb.ArtifactVersion{
						VersionId: 789,
					},
				},
				Type:          pb.Entity_ENTITY_ARTIFACTS,
				OwnershipData: map[string]int32{RepositoryIDEventKey: 456, ArtifactIDEventKey: 123},
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
			msg.Metadata.Set(GroupIDEventKey, fmt.Sprintf("%d", tt.args.groupID))
			msg.Metadata.Set(EntityTypeEventKey, tt.args.entType)
			msg.Metadata.Set(RepositoryIDEventKey, fmt.Sprintf("%d", tt.args.ownership["repository_id"]))
			if tt.args.entType == VersionedArtifactEventEntityType {
				msg.Metadata.Set(ArtifactIDEventKey, fmt.Sprintf("%d", tt.args.ownership["artifact_id"]))
			}

			got, err := parseEntityEvent(msg)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, got, "expected nil entity info")
				return
			}

			require.NoError(t, err, "unexpected error")
			require.NotNil(t, got, "expected non-nil entity info")

			//NOTE: Not checking the entity right now because it's a pain to compare
			assert.Equal(t, tt.want.GroupID, got.GroupID, "group id mismatch")
			assert.Equal(t, tt.want.Type, got.Type, "entity type mismatch")
			assert.Equal(t, tt.want.OwnershipData, got.OwnershipData, "ownership data mismatch")
		})
	}
}
