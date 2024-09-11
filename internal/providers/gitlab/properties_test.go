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

package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func Test_gitlabClient_GetEntityName(t *testing.T) {
	t.Parallel()

	type args struct {
		entityType minderv1.Entity
		props      *properties.Properties
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "nil properties",
			args: args{
				entityType: minderv1.Entity_ENTITY_REPOSITORIES,
				props:      nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "valid properties for repository succeeds",
			args: args{
				entityType: minderv1.Entity_ENTITY_REPOSITORIES,
				props: MustNewProperties(map[string]any{
					RepoPropertyGroupName:   "group",
					RepoPropertyProjectName: "project",
				}),
			},
			want:    "group/project",
			wantErr: false,
		},
		{
			name: "insufficient properties for repository fails (lacks project)",
			args: args{
				entityType: minderv1.Entity_ENTITY_REPOSITORIES,
				props: MustNewProperties(map[string]any{
					RepoPropertyGroupName: "group",
				}),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "insufficient properties for repository fails (lacks group)",
			args: args{
				entityType: minderv1.Entity_ENTITY_REPOSITORIES,
				props: MustNewProperties(map[string]any{
					RepoPropertyProjectName: "project",
				}),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "unsupported entity type fails",
			args: args{
				entityType: minderv1.Entity_ENTITY_UNSPECIFIED,
				props:      MustNewProperties(map[string]any{}),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &gitlabClient{}
			got, err := c.GetEntityName(tt.args.entityType, tt.args.props)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// MustNewProperties creates Properties from a map or panics
func MustNewProperties(props map[string]any) *properties.Properties {
	p, err := properties.NewProperties(props)
	if err != nil {
		panic(err)
	}
	return p
}
