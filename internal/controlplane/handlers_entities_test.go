// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	mockevents "github.com/stacklok/minder/internal/events/mock"
	mockgh "github.com/stacklok/minder/internal/providers/github/mock"
	mockmanager "github.com/stacklok/minder/internal/providers/manager/mock"
	rf "github.com/stacklok/minder/internal/repositories/github/mock/fixtures"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestServer_ReconcileEntityRegistration(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name             string
		RepoServiceSetup repoMockBuilder
		GitHubSetup      githubMockBuilder
		ProviderSetup    func(ctrl *gomock.Controller) *mockmanager.MockProviderManager
		EventerSetup     func(ctrl *gomock.Controller) *mockevents.MockInterface
		ProviderFails    bool
		ExpectedResults  []*pb.UpstreamRepositoryRef
		ExpectedError    string
	}{
		{
			Name:        "[positive] successful reconciliation",
			GitHubSetup: newGitHub(withSuccessfulListAllRepositories),
			RepoServiceSetup: rf.NewRepoService(
				rf.WithSuccessfulListRepositories(
					simpleDbRepository(repoName, remoteRepoId),
				),
			),
			ProviderSetup: func(ctrl *gomock.Controller) *mockmanager.MockProviderManager {
				providerManager := mockmanager.NewMockProviderManager(ctrl)
				providerManager.EXPECT().BulkInstantiateByTrait(
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(map[string]provinfv1.Provider{provider.Name: mockgh.NewMockGitHub(ctrl)}, []string{}, nil).Times(1)
				return providerManager
			},
			EventerSetup: func(ctrl *gomock.Controller) *mockevents.MockInterface {
				events := mockevents.NewMockInterface(ctrl)
				events.EXPECT().Publish(gomock.Any(), gomock.Any()).Times(1)
				return events
			},
		},
		{
			Name:        "[negative] failed to list repositories",
			GitHubSetup: newGitHub(withFailedListAllRepositories(errDefault)),
			RepoServiceSetup: rf.NewRepoService(
				rf.WithSuccessfulListRepositories(
					simpleDbRepository(repoName, remoteRepoId),
				),
			),
			ProviderSetup: func(ctrl *gomock.Controller) *mockmanager.MockProviderManager {
				providerManager := mockmanager.NewMockProviderManager(ctrl)
				providerManager.EXPECT().BulkInstantiateByTrait(
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(map[string]provinfv1.Provider{provider.Name: mockgh.NewMockGitHub(ctrl)}, []string{}, nil).Times(1)
				return providerManager
			},
			ExpectedError: "cannot register entities for providers: [github]",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := engine.WithEntityContext(context.Background(), &engine.EntityContext{
				Project: engine.Project{ID: projectID},
			})

			prov := scenario.GitHubSetup(ctrl)
			manager := mockmanager.NewMockProviderManager(ctrl)
			manager.EXPECT().BulkInstantiateByTrait(
				gomock.Any(),
				gomock.Eq(projectID),
				gomock.Eq(db.ProviderTypeRepoLister),
				gomock.Eq(""),
			).Return(map[string]provinfv1.Provider{provider.Name: prov}, []string{}, nil)

			server := createServer(
				ctrl,
				scenario.RepoServiceSetup,
				scenario.ProviderFails,
				manager,
			)

			if scenario.EventerSetup != nil {
				server.evt = scenario.EventerSetup(ctrl)
			}

			projectIDStr := projectID.String()
			req := &pb.ReconcileEntityRegistrationRequest{
				Context: &pb.Context{
					Project: &projectIDStr,
				},
			}
			res, err := server.ReconcileEntityRegistration(ctx, req)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.Nil(t, res)
				require.Contains(t, err.Error(), scenario.ExpectedError)
			}
		})
	}
}
