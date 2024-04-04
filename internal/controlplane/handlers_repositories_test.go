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
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/config/server"
	mockcrypto "github.com/stacklok/minder/internal/crypto/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/providers"
	mockgh "github.com/stacklok/minder/internal/providers/github/mock"
	ghprovider "github.com/stacklok/minder/internal/providers/github/oauth"
	"github.com/stacklok/minder/internal/providers/ratecache"
	ghrepo "github.com/stacklok/minder/internal/repositories/github"
	mockghrepo "github.com/stacklok/minder/internal/repositories/github/mock"
	"github.com/stacklok/minder/internal/util/ptr"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
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
			ExpectedError: "cannot retrieve provider",
		},
		{
			Name:          "Repo creation fails when repo name is missing",
			RepoOwner:     repoOwner,
			RepoName:      "",
			ExpectedError: "missing repository name",
		},
		{
			Name:             "Repo creation fails when repo does not exist in Github",
			RepoOwner:        repoOwner,
			RepoName:         repoName,
			RepoServiceSetup: newRepoService(withFailedCreate(errDefault)),
			ExpectedError:    errDefault.Error(),
		},
		{
			Name:             "Repo creation fails repo is private, and private repos are not allowed",
			RepoOwner:        repoOwner,
			RepoName:         repoName,
			RepoServiceSetup: newRepoService(withFailedCreate(ghrepo.ErrPrivateRepoForbidden)),
			ExpectedError:    "private repos cannot be registered in this project",
		},
		{
			Name:             "Repo creation on unexpected error",
			RepoOwner:        repoOwner,
			RepoName:         repoName,
			RepoServiceSetup: newRepoService(withFailedCreate(errDefault)),
			ExpectedError:    errDefault.Error(),
		},
		{
			Name:             "Repo creation is successful",
			RepoOwner:        repoOwner,
			RepoName:         repoName,
			RepoServiceSetup: newRepoService(withSuccessfulCreate),
		},
	}

	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := engine.WithEntityContext(context.Background(), &engine.EntityContext{
				Provider: engine.Provider{Name: ghprovider.Github},
				Project:  engine.Project{ID: projectID},
			})

			server := createServer(ctrl, scenario.RepoServiceSetup, newGitHub(), scenario.ProviderFails)

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

func TestServer_ListRemoteRepositoriesFromProvider(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name             string
		RepoServiceSetup repoMockBuilder
		GitHubSetup      githubMockBuilder
		ProviderFails    bool
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
		},
	}

	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := engine.WithEntityContext(context.Background(), &engine.EntityContext{
				Project: engine.Project{ID: projectID},
			})

			server := createServer(ctrl, scenario.RepoServiceSetup, scenario.GitHubSetup, scenario.ProviderFails)

			projectIDStr := projectID.String()
			req := &pb.ListRemoteRepositoriesFromProviderRequest{
				Context: &pb.Context{
					Project: &projectIDStr,
				},
			}
			res, err := server.ListRemoteRepositoriesFromProvider(ctx, req)
			if scenario.ExpectedError == "" {
				expectation := &pb.ListRemoteRepositoriesFromProviderResponse{
					Results: []*pb.UpstreamRepositoryRef{
						{
							Context: &pb.Context{
								Provider: &provider.Name,
								Project:  ptr.Ptr(projectID.String()),
							},
							Owner:  repoOwner,
							Name:   repoName,
							RepoId: remoteRepoId,
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
			Name:             "deletion fails when provider cannot be found",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: newRepoService(withSuccessfulGetRepoByName),
			ProviderFails:    true,
			ExpectedError:    "cannot retrieve provider",
		},
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
			Name:             "deletion fails when repo is not found",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: newRepoService(withFailedGetRepoByName(sql.ErrNoRows)),
			ExpectedError:    "repository not found",
		},
		{
			Name:             "deletion fails when repo service returns error",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: newRepoService(withSuccessfulGetRepoByName, withFailedDelete(errDefault)),
			ExpectedError:    "unexpected error deleting repo",
		},
		{
			Name:             "delete by ID fails when repo service returns error",
			RepoID:           repoID,
			RepoServiceSetup: newRepoService(withSuccessfulGetRepoById, withFailedDelete(errDefault)),
			ExpectedError:    "unexpected error deleting repo",
		},
		{
			Name:             "delete by name succeeds",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: newRepoService(withSuccessfulGetRepoByName, withSuccessfulDelete),
		},
		{
			Name:             "delete by ID succeeds",
			RepoID:           repoID,
			RepoServiceSetup: newRepoService(withSuccessfulGetRepoById, withSuccessfulDelete),
		},
	}

	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := engine.WithEntityContext(context.Background(), &engine.EntityContext{
				Provider: engine.Provider{Name: ghprovider.Github},
				Project:  engine.Project{ID: projectID},
			})

			server := createServer(ctrl, scenario.RepoServiceSetup, newGitHub(), scenario.ProviderFails)

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
	repoServiceMock   = *mockghrepo.MockRepositoryService
	repoMockBuilder   = func(*gomock.Controller) repoServiceMock
	githubMock        = *mockgh.MockGitHub
	githubMockBuilder = func(*gomock.Controller) githubMock
)

const (
	repoOwner        = "acme-corp"
	repoName         = "api-gateway"
	repoOwnerAndName = "acme-corp/api-gateway"
	repoID           = "3eb6d254-4163-460f-89f7-44e2ae916e71"
	remoteRepoId     = 123456
	accessToken      = "TOKEN"
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
	}
)

func newRepoService(opts ...func(repoServiceMock)) repoMockBuilder {
	return func(ctrl *gomock.Controller) repoServiceMock {
		mock := mockghrepo.NewMockRepositoryService(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
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

func withSuccessfulCreate(mock repoServiceMock) {
	mock.EXPECT().
		CreateRepository(gomock.Any(), gomock.Any(), gomock.Any(), projectID, repoOwner, repoName).
		Return(creationResult, nil)
}

func withFailedCreate(err error) func(repoServiceMock) {
	return func(mock repoServiceMock) {
		mock.EXPECT().
			CreateRepository(gomock.Any(), gomock.Any(), gomock.Any(), projectID, repoOwner, repoName).
			Return(nil, err)
	}
}

func withSuccessfulDelete(mock repoServiceMock) {
	withFailedDelete(nil)(mock)
}

func withFailedDelete(err error) func(repoServiceMock) {
	return func(mock repoServiceMock) {
		mock.EXPECT().
			DeleteRepository(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(err)
	}
}

func withSuccessfulGetRepoById(mock repoServiceMock) {
	mock.EXPECT().
		GetRepositoryById(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(db.Repository{}, nil)
}

func withSuccessfulGetRepoByName(mock repoServiceMock) {
	mock.EXPECT().
		GetRepositoryByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(db.Repository{}, nil)
}

func withFailedGetRepoByName(err error) func(repoServiceMock) {
	return func(mock repoServiceMock) {
		mock.EXPECT().
			GetRepositoryByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(db.Repository{}, err)
	}
}

func withSuccessfulListAllRepositories(mock githubMock) {
	mock.EXPECT().
		ListAllRepositories(gomock.Any()).
		Return([]*pb.Repository{&existingRepo}, nil)
}

func withFailedListAllRepositories(err error) func(githubMock) {
	return func(mock githubMock) {
		mock.EXPECT().
			ListAllRepositories(gomock.Any()).
			Return(nil, err)
	}
}

func createServer(ctrl *gomock.Controller, repoServiceSetup repoMockBuilder, githubSetup githubMockBuilder, providerFails bool) *Server {
	var svc ghrepo.RepositoryService
	if repoServiceSetup != nil {
		svc = repoServiceSetup(ctrl)
	}

	// stubs needed for providers to work
	// TODO: this provider logic should be better encapsulated from the controlplane
	mockCryptoEngine := mockcrypto.NewMockEngine(ctrl)
	mockCryptoEngine.EXPECT().
		DecryptOAuthToken(gomock.Any()).
		Return(oauth2.Token{AccessToken: accessToken}, nil).
		AnyTimes()
	cancelable, cancel := context.WithCancel(context.Background())
	clientCache := ratecache.NewRestClientCache(cancelable)
	defer cancel()

	var gh *mockgh.MockGitHub
	if githubSetup != nil {
		gh = githubSetup(ctrl)
	}

	clientCache.Set("", accessToken, db.ProviderTypeGithub, gh)

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
			FindProviders(gomock.Any(), gomock.Any()).
			Return([]db.Provider{}, errDefault)
	} else {
		store.EXPECT().
			FindProviders(gomock.Any(), gomock.Any()).
			Return([]db.Provider{provider}, nil).AnyTimes()
		store.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
			Return(db.ProviderAccessToken{
				EncryptedToken: "encryptedToken",
			}, nil).AnyTimes()
	}

	return &Server{
		store:           store,
		repos:           svc,
		cryptoEngine:    mockCryptoEngine,
		restClientCache: clientCache,
		cfg:             &server.Config{},
		providerStore:   providers.NewProviderStore(store),
	}
}
