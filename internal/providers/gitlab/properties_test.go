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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"

	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/providers/credentials"
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

func Test_gitlabClient_FetchAllProperties(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx        context.Context
		getByProps *properties.Properties
		entType    minderv1.Entity
	}

	tests := []struct {
		name                 string
		args                 args
		want                 *properties.Properties
		wantErr              bool
		gitLabServerMockFunc func(w http.ResponseWriter, r *http.Request)
	}{
		{
			name: "unsupported entity type",
			args: args{
				ctx:        context.TODO(),
				getByProps: &properties.Properties{},
				entType:    minderv1.Entity_ENTITY_UNSPECIFIED,
			},
			want:    nil,
			wantErr: true,
			gitLabServerMockFunc: func(_ http.ResponseWriter, _ *http.Request) {
				// No HTTP call needed for unsupported entity type
			},
		},
		{
			name: "repository succeeds",
			args: args{
				ctx: context.TODO(),
				getByProps: MustNewProperties(map[string]any{
					properties.PropertyUpstreamID: "1",
				}),
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
			},
			want: MustNewProperties(map[string]any{
				properties.RepoPropertyIsPrivate:  true,
				properties.RepoPropertyIsArchived: false,
				properties.RepoPropertyIsFork:     false,
			}),
			wantErr: false,
			gitLabServerMockFunc: func(w http.ResponseWriter, _ *http.Request) {
				resp := &gitlab.Project{
					ID:                1,
					Name:              "project-1",
					Description:       "project-1 description",
					Visibility:        gitlab.PrivateVisibility,
					Archived:          false,
					ForkedFromProject: nil,
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				//nolint:gosec // This is a test
				json.NewEncoder(w).Encode(resp)
			},
		},
		{
			name: "repository not found",
			args: args{
				ctx: context.TODO(),
				getByProps: MustNewProperties(map[string]any{
					properties.PropertyUpstreamID: "1",
				}),
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
			},
			want:    nil,
			wantErr: true,
			gitLabServerMockFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
			},
		},
		{
			name: "repository fails",
			args: args{
				ctx: context.TODO(),
				getByProps: MustNewProperties(map[string]any{
					properties.PropertyUpstreamID: "1",
				}),
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
			},
			want:    nil,
			wantErr: true,
			gitLabServerMockFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
			},
		},
		{
			name: "repository fails to decode",
			args: args{
				ctx: context.TODO(),
				getByProps: MustNewProperties(map[string]any{
					properties.PropertyUpstreamID: "1",
				}),
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
			},
			want:    nil,
			wantErr: true,
			gitLabServerMockFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				//nolint:gosec // This is a test
				w.Write([]byte("invalid json"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(tt.gitLabServerMockFunc))
			defer ts.Close()

			gitlabClient := newTestGitlabProvider(ts.URL)

			got, err := gitlabClient.FetchAllProperties(tt.args.ctx, tt.args.getByProps, tt.args.entType, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for wkey, wp := range tt.want.Iterate() {
					gp := got.GetProperty(wkey)
					assert.NotNil(t, gp, "property %s not found", wkey)
					if gp == nil {
						if wp == nil {
							continue
						}
					}
					assert.Equal(t, wp.RawValue(), gp.RawValue())
				}
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

func newTestGitlabProvider(endpoint string) *gitlabClient {
	return &gitlabClient{
		cred: &credentials.GitLabTokenCredential{},
		glcfg: &minderv1.GitLabProviderConfig{
			Endpoint: endpoint,
		},
		cli: &http.Client{},
	}
}
