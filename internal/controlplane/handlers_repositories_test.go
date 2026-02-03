// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/entities/models"
	propService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers"
	ghprovider "github.com/mindersec/minder/internal/providers/github/clients"
	mockgh "github.com/mindersec/minder/internal/providers/github/mock"
	ghprops "github.com/mindersec/minder/internal/providers/github/properties"
	"github.com/mindersec/minder/internal/providers/manager"
	mockmanager "github.com/mindersec/minder/internal/providers/manager/mock"
	reposvc "github.com/mindersec/minder/internal/repositories"
	mockrepo "github.com/mindersec/minder/internal/repositories/mock"
	rf "github.com/mindersec/minder/internal/repositories/mock/fixtures"
	"github.com/mindersec/minder/internal/util/ptr"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/entities/properties"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestServer_RegisterRepository(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name             string
		RepoOwner        string
		RepoName         string
		RepoServiceSetup repoMockBuilder
		ProviderFails    bool
		ExpectedError    string
	}{
		{
			Name:          "Repo creation fails when provider cannot be found",
			RepoOwner:     repoOwner,
			RepoName:      repoName,
			ProviderFails: true,
			ExpectedError: "cannot get provider",
		},
		{
			Name:          "Repo creation fails when repo name is missing",
			RepoOwner:     repoOwner,
			RepoName:      "",
			ExpectedError: "missing repository name",
		},
		{
			Name:      "Repo creation fails when repo does not exist in Github",
			RepoOwner: repoOwner,
			RepoName:  repoName,
			RepoServiceSetup: rf.NewRepoService(
				rf.WithFailedCreate(
					errDefault,
					projectID,
				)),
			ExpectedError: errDefault.Error(),
		},
		{
			Name:      "Repo creation fails repo is private, and private repos are not allowed",
			RepoOwner: repoOwner,
			RepoName:  repoName,
			RepoServiceSetup: rf.NewRepoService(rf.WithFailedCreate(
				reposvc.ErrPrivateRepoForbidden,
				projectID,
			)),
			ExpectedError: "private repositories are not allowed in this project",
		},
		{
			Name:      "Repo creation fails repo is archived, and archived repos are not allowed",
			RepoOwner: repoOwner,
			RepoName:  repoName,
			RepoServiceSetup: rf.NewRepoService(rf.WithFailedCreate(
				reposvc.ErrArchivedRepoForbidden,
				projectID,
			)),
			ExpectedError: "archived repositories cannot be registered",
		},
		{
			Name:      "Repo creation on unexpected error",
			RepoOwner: repoOwner,
			RepoName:  repoName,
			RepoServiceSetup: rf.NewRepoService(rf.WithFailedCreate(
				errDefault,
				projectID,
			)),
			ExpectedError: errDefault.Error(),
		},
		{
			Name:      "Repo creation is successful",
			RepoOwner: repoOwner,
			RepoName:  repoName,
			RepoServiceSetup: rf.NewRepoService(rf.WithSuccessfulCreate(
				projectID,
				creationResult,
			)),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Provider: engcontext.Provider{Name: ghprovider.Github},
				Project:  engcontext.Project{ID: projectID},
			})

			server := createServer(
				ctrl,
				scenario.RepoServiceSetup,
				scenario.ProviderFails,
				nil,
			)

			req := &pb.RegisterRepositoryRequest{
				Repository: &pb.UpstreamRepositoryRef{
					Owner: scenario.RepoOwner,
					Name:  scenario.RepoName,
				},
			}
			res, err := server.RegisterRepository(ctx, req)
			if scenario.ExpectedError == "" {
				expectation := &pb.RegisterRepositoryResponse{
					Result: &pb.RegisterRepoResult{
						Repository: creationResult,
						Status: &pb.RegisterRepoResult_Status{
							Success: true,
						},
					},
				}
				require.NoError(t, err)
				require.Equal(t, res, expectation)
			} else {
				require.Nil(t, res)
				require.Contains(t, err.Error(), scenario.ExpectedError)
			}
		})
	}
}

func TestServer_ListRepositories(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name             string
		RepoServiceSetup repoMockBuilder
		ProviderSetup    func(ctrl *gomock.Controller, mgr *mockmanager.MockProviderManager)
		ProviderFails    bool
		ExpectedResults  []*pb.Repository
		ExpectedError    string
	}{
		{
			Name: "List repositories succeeds with multiple results",
			RepoServiceSetup: rf.NewRepoService(
				rf.WithSuccessfulListRepositories(
					simpleDbRepository(repoName, remoteRepoId),
					simpleDbRepository(repoName2, remoteRepoId2),
				),
			),
			ProviderSetup: func(ctrl *gomock.Controller, mgr *mockmanager.MockProviderManager) {
				prov := newGitHub(func(mock githubMock) {
					first := mock.EXPECT().PropertiesToProtoMessage(pb.Entity_ENTITY_REPOSITORIES, gomock.Any()).
						Return(&pb.Repository{
							Name:   repoName,
							RepoId: remoteRepoId,
							Properties: mustNewPBStruct(map[string]any{
								"github/repo_id": map[string]any{
									"minder.internal.type": "int64",
									"value":                strconv.FormatInt(remoteRepoId, 10),
								},
							}),
						}, nil)
					mock.EXPECT().PropertiesToProtoMessage(pb.Entity_ENTITY_REPOSITORIES, gomock.Any()).
						Return(&pb.Repository{
							Name:   repoName2,
							RepoId: remoteRepoId2,
							Properties: mustNewPBStruct(map[string]any{
								"github/repo_id": map[string]any{
									"minder.internal.type": "int64",
									"value":                strconv.FormatInt(remoteRepoId2, 10),
								},
							}),
						}, nil).After(first)

				})(ctrl)
				mgr.EXPECT().InstantiateFromID(gomock.Any(), uuid.Nil).Return(prov, nil).Times(2)
			},
			ExpectedResults: []*pb.Repository{
				{
					Name: repoName,
					Context: &pb.Context{
						Provider: &provider.Name,
						Project:  ptr.Ptr(projectID.String()),
					},
					RepoId: remoteRepoId,
					Properties: mustNewPBStruct(map[string]any{
						"github/repo_id": map[string]any{
							"minder.internal.type": "int64",
							"value":                strconv.FormatInt(remoteRepoId, 10),
						}}),
				},
				{
					Name: repoName2,
					Context: &pb.Context{
						Provider: &provider.Name,
						Project:  ptr.Ptr(projectID.String()),
					},
					RepoId: remoteRepoId2,
					Properties: mustNewPBStruct(map[string]any{
						"github/repo_id": map[string]any{
							"minder.internal.type": "int64",
							"value":                strconv.FormatInt(remoteRepoId2, 10),
						},
					}),
				},
			},
		},
		{
			Name:          "List repositories fails when provider cannot be found",
			ProviderFails: true,
			ExpectedError: "cannot retrieve providers",
		},
		{
			Name: "List repositories succeeds with empty results",
			RepoServiceSetup: rf.NewRepoService(
				rf.WithSuccessfulListRepositories(),
			),
			ExpectedResults: nil,
		},
		{
			Name: "List repositories fails when repo service returns error",
			RepoServiceSetup: rf.NewRepoService(
				rf.WithFailedListRepositories(errDefault),
			),
			ExpectedError: errDefault.Error(),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Provider: engcontext.Provider{Name: ghprovider.Github},
				Project:  engcontext.Project{ID: projectID},
			})

			mgr := mockmanager.NewMockProviderManager(ctrl)
			if scenario.ProviderSetup != nil {
				scenario.ProviderSetup(ctrl, mgr)
			}

			server := createServer(
				ctrl,
				scenario.RepoServiceSetup,
				scenario.ProviderFails,
				mgr,
			)

			req := &pb.ListRepositoriesRequest{}
			res, err := server.ListRepositories(ctx, req)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Len(t, res.Results, len(scenario.ExpectedResults))
				require.Empty(t, res.Cursor)

				diff := cmp.Diff(scenario.ExpectedResults, res.Results, protocmp.Transform())
				if diff != "" {
					t.Errorf("repository mismatch (-want +got):\n%s", diff)
				}
			} else {
				require.Nil(t, res)
				require.Contains(t, err.Error(), scenario.ExpectedError)
			}
		})
	}
}

func TestServer_ListRemoteRepositoriesFromProvider(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name             string
		RepoServiceSetup repoMockBuilder
		GitHubSetup      githubMockBuilder
		ProviderFails    bool
		ExpectedResults  []*UpstreamRepoAndEntityRef
		ExpectedError    string
	}{
		{
			Name:          "List remote repositories fails when all providers error",
			GitHubSetup:   newGitHub(withFailedListAllRepositories(errDefault)),
			ExpectedError: "cannot list repositories for providers: [github]",
		},
		{
			Name:        "List remote repositories succeeds when all providers succeed",
			GitHubSetup: newGitHub(withSuccessfulListAllRepositories),
			RepoServiceSetup: rf.NewRepoService(
				rf.WithSuccessfulListRepositories(
					simpleDbRepository(repoName, remoteRepoId),
				),
			),
			ExpectedResults: []*UpstreamRepoAndEntityRef{
				simpleUpstreamRepositoryRef(repoName, remoteRepoId, true),
				simpleUpstreamRepositoryRef(repoName2, remoteRepoId2, false),
			},
		},
		{
			Name:        "List remote repositories fails when db fails",
			GitHubSetup: newGitHub(withSuccessfulListAllRepositories),
			RepoServiceSetup: rf.NewRepoService(
				rf.WithFailedListRepositories(errors.New("oops")),
			),
			ExpectedError: "cannot list repositories for providers: [github]",
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

			prov := scenario.GitHubSetup(ctrl)
			mgr := mockmanager.NewMockProviderManager(ctrl)
			mgr.EXPECT().BulkInstantiateByTrait(
				gomock.Any(),
				gomock.Eq(projectID),
				gomock.Eq(db.ProviderTypeRepoLister),
				gomock.Eq(""),
			).Return(map[uuid.UUID]manager.NameProviderTuple{
				provider.ID: {
					Name:     provider.Name,
					Provider: prov,
				},
			}, []string{}, nil)

			server := createServer(
				ctrl,
				scenario.RepoServiceSetup,
				scenario.ProviderFails,
				mgr,
			)

			projectIDStr := projectID.String()
			req := &pb.ListRemoteRepositoriesFromProviderRequest{
				Context: &pb.Context{
					Project: &projectIDStr,
				},
			}
			res, err := server.ListRemoteRepositoriesFromProvider(ctx, req)
			if scenario.ExpectedError == "" {
				expectation := &pb.ListRemoteRepositoriesFromProviderResponse{}
				for _, repo := range scenario.ExpectedResults {
					expectation.Results = append(expectation.Results, repo.Repo)
					expectation.Entities = append(expectation.Entities, repo.Entity)
				}
				require.NoError(t, err)

				require.Len(t, res.Results, len(scenario.ExpectedResults))
				require.Len(t, res.Entities, len(scenario.ExpectedResults))

				require.Equal(t, expectation.Results, res.Results)
				// we can't compare the structs directly because the properties are not guaranteed to be in the same order
				// and the structpb wrapper converts some values as internal representations
				for i := range res.Entities {
					require.Equal(t, expectation.Entities[i].Entity.Type, res.Entities[i].Entity.Type)
					require.Equal(t, expectation.Entities[i].Registered, res.Entities[i].Registered)
				}
			} else {
				require.Nil(t, res)
				require.Contains(t, err.Error(), scenario.ExpectedError)
			}
		})
	}
}

// lump both deletion endpoints together since they are so similar
func TestServer_DeleteRepository(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name             string
		RepoName         string
		RepoID           string
		RepoServiceSetup repoMockBuilder
		ProviderFails    bool
		ExpectedError    string
	}{
		{
			Name:          "delete by name fails when name is malformed",
			RepoName:      "I am not a repo name",
			ExpectedError: "invalid repository name",
		},
		{
			Name:          "delete by ID fails when ID is malformed",
			RepoID:        "I am not a UUID",
			ExpectedError: "invalid repository ID",
		},
		{
			Name:             "deletion fails when repo service returns error",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: rf.NewRepoService(rf.WithFailedDeleteByName(errDefault)),
			ExpectedError:    "unexpected error deleting repo",
		},
		{
			Name:             "delete by ID fails when repo service returns error",
			RepoID:           repoID,
			RepoServiceSetup: rf.NewRepoService(rf.WithFailedDeleteByID(errDefault)),
			ExpectedError:    "unexpected error deleting repo",
		},
		{
			Name:             "deletion fails when repo is not found",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: rf.NewRepoService(rf.WithFailedDeleteByName(sql.ErrNoRows)),
			ExpectedError:    "repository not found",
		},
		{
			Name:             "delete by ID fails when repo is not found",
			RepoID:           repoID,
			RepoServiceSetup: rf.NewRepoService(rf.WithFailedDeleteByID(sql.ErrNoRows)),
			ExpectedError:    "repository not found",
		},
		{
			Name:             "delete by name succeeds",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: rf.NewRepoService(rf.WithSuccessfulDeleteByName()),
		},
		{
			Name:             "delete by ID succeeds",
			RepoID:           repoID,
			RepoServiceSetup: rf.NewRepoService(rf.WithSuccessfulDeleteByID()),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Provider: engcontext.Provider{Name: ghprovider.Github},
				Project:  engcontext.Project{ID: projectID},
			})

			server := createServer(
				ctrl,
				scenario.RepoServiceSetup,
				scenario.ProviderFails,
				nil,
			)

			var result string
			var resultError error
			var expectation string
			if scenario.RepoName != "" {
				req := &pb.DeleteRepositoryByNameRequest{
					Name: scenario.RepoName,
				}
				res, err := server.DeleteRepositoryByName(ctx, req)
				if res != nil {
					result = res.Name
					expectation = scenario.RepoName
				}
				resultError = err
			} else {
				req := &pb.DeleteRepositoryByIdRequest{
					RepositoryId: scenario.RepoID,
				}
				res, err := server.DeleteRepositoryById(ctx, req)
				if res != nil {
					result = res.RepositoryId
					expectation = scenario.RepoID
				}
				resultError = err
			}

			if scenario.ExpectedError == "" {
				require.NoError(t, resultError)
				require.Equal(t, result, expectation)
			} else {
				require.Empty(t, result)
				require.ErrorContains(t, resultError, scenario.ExpectedError)
			}
		})
	}
}

type (
	repoServiceMock   = *mockrepo.MockRepositoryService
	repoMockBuilder   = func(*gomock.Controller) repoServiceMock
	githubMock        = *mockgh.MockGitHub
	githubMockBuilder = func(*gomock.Controller) githubMock
)

const (
	repoOwner              = "acme-corp"
	repoName               = "api-gateway"
	repoOwnerAndName       = "acme-corp/api-gateway"
	repoID                 = "3eb6d254-4163-460f-89f7-44e2ae916e71"
	remoteRepoId     int64 = 123456
	repoName2              = "another-repo"
	remoteRepoId2    int64 = 234567
	accessToken            = "TOKEN"
)

var (
	projectID  = uuid.New()
	errDefault = errors.New("oh no")
	provider   = db.Provider{
		ID:         uuid.UUID{},
		Name:       ghprovider.Github,
		Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeRepoLister},
		Version:    provinfv1.V1,
	}
	creationResult = &pb.Repository{
		Owner: repoOwner,
		Name:  repoName,
	}
	existingRepo = pb.Repository{
		Owner:  repoOwner,
		Name:   repoName,
		RepoId: remoteRepoId,
		Properties: mustNewPBStruct(map[string]any{
			properties.PropertyUpstreamID: fmt.Sprintf("%d", remoteRepoId),
			ghprops.RepoPropertyId:        remoteRepoId,
			ghprops.RepoPropertyName:      repoName,
			ghprops.RepoPropertyOwner:     repoOwner,
		}),
	}
	existingRepo2 = pb.Repository{
		Owner:  repoOwner,
		Name:   repoName2,
		RepoId: 234567,
		Properties: mustNewPBStruct(map[string]any{
			properties.PropertyUpstreamID: "234567",
			ghprops.RepoPropertyId:        234567,
			ghprops.RepoPropertyName:      repoName2,
			ghprops.RepoPropertyOwner:     repoOwner,
		}),
	}
)

func mustNewPBStruct(m map[string]any) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		panic(err)
	}
	return s
}

func simpleDbRepository(name string, id int64) *models.EntityWithProperties {
	//nolint:errcheck // this shouldn't fail
	props := properties.NewProperties(map[string]any{
		"repo_id":                     id,
		properties.PropertyUpstreamID: fmt.Sprintf("%d", id),
	})
	return models.NewEntityWithPropertiesFromInstance(models.EntityInstance{
		ID:   uuid.UUID{},
		Type: pb.Entity_ENTITY_REPOSITORIES,
		Name: name,
	}, props)
}

func simpleUpstreamRepositoryRef(name string, id int64, registered bool) *UpstreamRepoAndEntityRef {
	props := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: fmt.Sprintf("%d", id),
		ghprops.RepoPropertyId:        id,
		ghprops.RepoPropertyName:      name,
		ghprops.RepoPropertyOwner:     repoOwner,
	})

	return &UpstreamRepoAndEntityRef{
		Repo: &pb.UpstreamRepositoryRef{
			Context: &pb.Context{
				Provider: &provider.Name,
				Project:  ptr.Ptr(projectID.String()),
			},
			Owner:      repoOwner,
			Name:       name,
			RepoId:     id,
			Registered: registered,
		},
		Entity: &pb.RegistrableUpstreamEntityRef{
			Entity: &pb.UpstreamEntityRef{
				Context: &pb.ContextV2{
					Provider:  provider.Name,
					ProjectId: projectID.String(),
				},
				Type:       pb.Entity_ENTITY_REPOSITORIES,
				Properties: props.ToProtoStruct(),
			},
			Registered: registered,
		},
	}
}

func newGitHub(opts ...func(mock githubMock)) githubMockBuilder {
	return func(ctrl *gomock.Controller) githubMock {
		mock := mockgh.NewMockGitHub(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func withSuccessfulListAllRepositories(mock githubMock) {
	mock.EXPECT().
		ListAllRepositories(gomock.Any()).
		Return([]*pb.Repository{&existingRepo, &existingRepo2}, nil)
}

func withFailedListAllRepositories(err error) func(githubMock) {
	return func(mock githubMock) {
		mock.EXPECT().
			ListAllRepositories(gomock.Any()).
			Return(nil, err)
	}
}

func createServer(
	ctrl *gomock.Controller,
	repoServiceSetup repoMockBuilder,
	providerFails bool,
	providerManager manager.ProviderManager,
) *Server {
	var svc reposvc.RepositoryService
	if repoServiceSetup != nil {
		svc = repoServiceSetup(ctrl)
	}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		GetParentProjects(gomock.Any(), projectID).
		Return([]uuid.UUID{projectID}, nil).
		AnyTimes()
	store.EXPECT().
		GetFeatureInProject(gomock.Any(), gomock.Any()).
		Return(json.RawMessage{}, nil).
		AnyTimes()

	if providerFails {
		store.EXPECT().
			GetProviderByID(gomock.Any(), gomock.Any()).
			Return(db.Provider{}, errDefault).AnyTimes()
		store.EXPECT().
			FindProviders(gomock.Any(), gomock.Any()).
			Return([]db.Provider{}, errDefault).AnyTimes()
	} else {
		store.EXPECT().
			GetProviderByID(gomock.Any(), gomock.Any()).
			Return(provider, nil).AnyTimes()
		store.EXPECT().
			FindProviders(gomock.Any(), gomock.Any()).
			Return([]db.Provider{provider}, nil).AnyTimes()
		store.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
			Return(db.ProviderAccessToken{
				EncryptedAccessToken: pqtype.NullRawMessage{
					Valid:      true,
					RawMessage: make(json.RawMessage, 16),
				},
			}, nil).AnyTimes()
	}

	return &Server{
		store:           store,
		repos:           svc,
		cfg:             &server.Config{},
		providerStore:   providers.NewProviderStore(store),
		providerManager: providerManager,
		props:           propService.NewPropertiesService(store),
	}
}
