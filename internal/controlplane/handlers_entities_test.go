// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	mockevents "github.com/mindersec/minder/internal/events/mock"
	mockgh "github.com/mindersec/minder/internal/providers/github/mock"
	"github.com/mindersec/minder/internal/providers/manager"
	mockmanager "github.com/mindersec/minder/internal/providers/manager/mock"
	rf "github.com/mindersec/minder/internal/repositories/mock/fixtures"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestServer_ReconcileEntityRegistration(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name             string
		RepoServiceSetup repoMockBuilder
		GitHubSetup      githubMockBuilder
		ProviderSetup    func(ctrl *gomock.Controller) *mockmanager.MockProviderManager
		EventerSetup     func(ctrl *gomock.Controller) *mockevents.MockInterface
		EntityType       pb.Entity
		ProviderFails    bool
		ExpectedResults  []*pb.UpstreamRepositoryRef
		ExpectedError    string
	}{
		{
			Name:        "[positive] successful reconciliation",
			EntityType:  pb.Entity_ENTITY_REPOSITORIES,
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
				).Return(map[uuid.UUID]manager.NameProviderTuple{
					uuid.New(): {
						Name:     provider.Name,
						Provider: mockgh.NewMockGitHub(ctrl),
					},
				}, []string{}, nil).Times(1)
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
			EntityType:  pb.Entity_ENTITY_REPOSITORIES,
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
				).Return(map[uuid.UUID]manager.NameProviderTuple{
					uuid.New(): {
						Name:     provider.Name,
						Provider: mockgh.NewMockGitHub(ctrl),
					},
				}, []string{}, nil).Times(1)
				return providerManager
			},
			ExpectedError: "cannot register entities for providers: [github]",
		},
		{
			Name:          "[negative] unexpected entity type",
			EntityType:    pb.Entity_ENTITY_ARTIFACTS,
			ExpectedError: "entity type artifact not supported",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			})

			mgr := mockmanager.NewMockProviderManager(ctrl)
			if scenario.ProviderSetup != nil && scenario.GitHubSetup != nil {
				prov := scenario.GitHubSetup(ctrl)
				mgr.EXPECT().BulkInstantiateByTrait(
					gomock.Any(),
					gomock.Eq(projectID),
					gomock.Eq(db.ProviderTypeRepoLister),
					gomock.Eq(""),
				).Return(map[uuid.UUID]manager.NameProviderTuple{
					uuid.New(): {
						Name:     provider.Name,
						Provider: prov,
					},
				}, []string{}, nil)
			}

			server := createServer(
				ctrl,
				scenario.RepoServiceSetup,
				scenario.ProviderFails,
				mgr,
			)

			if scenario.EventerSetup != nil {
				server.evt = scenario.EventerSetup(ctrl)
			}

			projectIDStr := projectID.String()
			req := &pb.ReconcileEntityRegistrationRequest{
				Context: &pb.Context{
					Project: &projectIDStr,
				},
				Entity: scenario.EntityType.ToString(),
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
