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
	"errors"
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v56/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/stacklok/minder/database/mock"
	mockcrypto "github.com/stacklok/minder/internal/crypto/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"

	ghprovider "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/ratecache"
	ghrepo "github.com/stacklok/minder/internal/repositories/github"
	mockghrepo "github.com/stacklok/minder/internal/repositories/github/mock"
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
			ExpectedError: "cannot retrieve providers",
		},
		{
			Name:          "Repo creation fails when repo name is missing",
			RepoOwner:     repoOwner,
			RepoName:      "",
			ExpectedError: "missing repository owner and/or name",
		},
		{
			Name:          "Repo creation fails when repo owner is missing",
			RepoOwner:     "",
			RepoName:      repoName,
			ExpectedError: "missing repository owner and/or name",
		},
		{
			Name:             "Repo creation fails when repo does not exist in Github",
			RepoOwner:        repoOwner,
			RepoName:         repoName,
			RepoServiceSetup: newRepoService(withFailedCreate(ghprovider.ErrNotFound)),
			ExpectedError:    ghprovider.ErrNotFound.Error(),
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

			server := createServer(ctrl, scenario.RepoServiceSetup, scenario.ProviderFails)

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
				// due to the mix of error handling styles in this endpoint, we
				// need to do some hackery here
				var errMsg string
				if err != nil {
					errMsg = err.Error()
				} else if err == nil && res.Result.Status.Success == false && res.Result.Status.Error != nil {
					errMsg = *res.Result.Status.Error
				} else {
					t.Fatal("expected error, but no error was found")
				}
				require.True(t, res == nil || res.Result.Status.Success == false)
				require.Contains(t, errMsg, scenario.ExpectedError)
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
			Name:          "deletion fails when provider cannot be found",
			RepoName:      repoOwnerAndName,
			ProviderFails: true,
			ExpectedError: "cannot retrieve providers",
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
			RepoServiceSetup: newRepoService(withFailedDeleteByName(sql.ErrNoRows)),
			ExpectedError:    "repository not found",
		},
		{
			Name:             "deletion fails when repo service returns error",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: newRepoService(withFailedDeleteByName(errDefault)),
			ExpectedError:    "unexpected error deleting repo",
		},
		{
			Name:             "delete by ID fails when repo service returns error",
			RepoID:           repoID,
			RepoServiceSetup: newRepoService(withFailedDeleteByID(errDefault)),
			ExpectedError:    "unexpected error deleting repo",
		},
		{
			Name:             "delete by name succeeds",
			RepoName:         repoOwnerAndName,
			RepoServiceSetup: newRepoService(withSuccessfulDeleteByName),
		},
		{
			Name:             "delete by ID succeeds",
			RepoID:           repoID,
			RepoServiceSetup: newRepoService(withSuccessfulDeleteByID),
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

			server := createServer(ctrl, scenario.RepoServiceSetup, scenario.ProviderFails)

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
	repoServiceMock = *mockghrepo.MockRepositoryService
	repoMockBuilder = func(*gomock.Controller) repoServiceMock
)

const (
	repoOwner        = "acme-corp"
	repoName         = "api-gateway"
	repoOwnerAndName = "acme-corp/api-gateway"
	repoID           = "3eb6d254-4163-460f-89f7-44e2ae916e71"
	accessToken      = "TOKEN"
)

var (
	projectID      = uuid.New()
	errDefault     = errors.New("oh no")
	creationResult = &pb.Repository{
		Owner: repoOwner,
		Name:  repoName,
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

func withSuccessfulCreate(mock repoServiceMock) {
	mock.EXPECT().
		CreateRepository(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(creationResult, nil)
}

func withFailedCreate(err error) func(repoServiceMock) {
	return func(mock repoServiceMock) {
		mock.EXPECT().
			CreateRepository(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, err)
	}
}

func withSuccessfulDeleteByName(mock repoServiceMock) {
	withFailedDeleteByName(nil)(mock)
}

func withFailedDeleteByName(err error) func(repoServiceMock) {
	return func(mock repoServiceMock) {
		mock.EXPECT().
			DeleteRepositoryByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(err)
	}
}

func withSuccessfulDeleteByID(mock repoServiceMock) {
	withFailedDeleteByID(nil)(mock)
}

func withFailedDeleteByID(err error) func(repoServiceMock) {
	return func(mock repoServiceMock) {
		mock.EXPECT().
			DeleteRepositoryByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(err)
	}
}

func createServer(
	ctrl *gomock.Controller,
	repoServiceSetup repoMockBuilder,
	providerFails bool,
) *Server {
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
	clientCache.Set("", accessToken, db.ProviderTypeGithub, &stubGitHub{})

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		GetParentProjects(gomock.Any(), projectID).
		Return([]uuid.UUID{projectID}, nil).
		AnyTimes()

	if providerFails {
		store.EXPECT().
			ListProvidersByProjectID(gomock.Any(), []uuid.UUID{projectID}).
			Return(nil, errDefault)
	} else {
		store.EXPECT().
			ListProvidersByProjectID(gomock.Any(), []uuid.UUID{projectID}).
			Return([]db.Provider{{
				ID:         uuid.New(),
				Name:       "github",
				Implements: []db.ProviderType{db.ProviderTypeGithub},
				Version:    provinfv1.V1,
			}}, nil).AnyTimes()
		store.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
			Return(db.ProviderAccessToken{
				EncryptedToken: "encryptedToken",
			}, nil).AnyTimes()
	}

	return &Server{
		store:           store,
		repoService:     svc,
		cryptoEngine:    mockCryptoEngine,
		restClientCache: clientCache,
	}
}

// Unfortunately it will not be possible to get rid of this until we decouple
// the provider logic from the controlplane (see comments in test cases)
type stubGitHub struct{}

// FakeGitHub implements v1.GitHub.  Holy wide interface, Batman!
var _ provinfv1.GitHub = (*stubGitHub)(nil)

// CloseSecurityAdvisory implements v1.GitHub.
func (*stubGitHub) CloseSecurityAdvisory(context.Context, string, string, string) error {
	panic("unimplemented")
}

// CreateHook implements v1.GitHub.
func (_ *stubGitHub) CreateHook(_ context.Context, _ string, _ string, _ *github.Hook) (*github.Hook, error) {
	panic("unimplemented")
}

// CreatePullRequest implements v1.GitHub.
func (*stubGitHub) CreatePullRequest(context.Context, string, string, string, string, string, string) (*github.PullRequest, error) {
	panic("unimplemented")
}

// CreateReview implements v1.GitHub.
func (*stubGitHub) CreateReview(context.Context, string, string, int, *github.PullRequestReviewRequest) (*github.PullRequestReview, error) {
	panic("unimplemented")
}

// CreateSecurityAdvisory implements v1.GitHub.
func (*stubGitHub) CreateSecurityAdvisory(context.Context, string, string, string, string, string, []*github.AdvisoryVulnerability) (string, error) {
	panic("unimplemented")
}

// DeleteHook implements v1.GitHub.
func (_ *stubGitHub) DeleteHook(_ context.Context, _ string, _ string, _ int64) (*github.Response, error) {
	panic("unimplemented")
}

// DismissReview implements v1.GitHub.
func (*stubGitHub) DismissReview(context.Context, string, string, int, int64, *github.PullRequestReviewDismissalRequest) (*github.PullRequestReview, error) {
	panic("unimplemented")
}

// Do implements v1.GitHub.
func (*stubGitHub) Do(context.Context, *http.Request) (*http.Response, error) {
	panic("unimplemented")
}

// GetBaseURL implements v1.GitHub.
func (*stubGitHub) GetBaseURL() string {
	panic("unimplemented")
}

// GetBranchProtection implements v1.GitHub.
func (*stubGitHub) GetBranchProtection(context.Context, string, string, string) (*github.Protection, error) {
	panic("unimplemented")
}

// GetOwner implements v1.GitHub.
func (*stubGitHub) GetOwner() string {
	panic("unimplemented")
}

// GetPackageByName implements v1.GitHub.
func (*stubGitHub) GetPackageByName(context.Context, bool, string, string, string) (*github.Package, error) {
	panic("unimplemented")
}

// GetPackageVersionById implements v1.GitHub.
func (*stubGitHub) GetPackageVersionById(context.Context, bool, string, string, string, int64) (*github.PackageVersion, error) {
	panic("unimplemented")
}

// GetPackageVersionByTag implements v1.GitHub.
func (*stubGitHub) GetPackageVersionByTag(context.Context, bool, string, string, string, string) (*github.PackageVersion, error) {
	panic("unimplemented")
}

// GetPackageVersions implements v1.GitHub.
func (*stubGitHub) GetPackageVersions(context.Context, bool, string, string, string) ([]*github.PackageVersion, error) {
	panic("unimplemented")
}

// GetPullRequest implements v1.GitHub.
func (*stubGitHub) GetPullRequest(context.Context, string, string, int) (*github.PullRequest, error) {
	panic("unimplemented")
}

func (*stubGitHub) CreateIssueComment(context.Context, string, string, int, string) (*github.IssueComment, error) {
	panic("unimplemented")
}

func (*stubGitHub) ListIssueComments(context.Context, string, string, int, *github.IssueListCommentsOptions) ([]*github.IssueComment, error) {
	panic("unimplemented")
}

func (*stubGitHub) UpdateIssueComment(context.Context, string, string, int64, string) error {
	panic("unimplemented")
}

func (*stubGitHub) UpdateReview(context.Context, string, string, int, int64, string) (*github.PullRequestReview, error) {
	panic("unimplemented")
}

// GetRepository implements v1.GitHub.
func (*stubGitHub) GetRepository(_ context.Context, _ string, _ string) (*github.Repository, error) {
	panic("unimplemented")
}

// GetToken implements v1.GitHub.
func (*stubGitHub) GetToken() string {
	panic("unimplemented")
}

// ListAllPackages implements v1.GitHub.
func (*stubGitHub) ListAllPackages(context.Context, bool, string, string, int, int) ([]*github.Package, error) {
	panic("unimplemented")
}

// ListAllRepositories implements v1.GitHub.
func (*stubGitHub) ListAllRepositories(context.Context, bool, string) ([]*github.Repository, error) {
	panic("unimplemented")
}

// ListFiles implements v1.GitHub.
func (*stubGitHub) ListFiles(context.Context, string, string, int, int, int) ([]*github.CommitFile, *github.Response, error) {
	panic("unimplemented")
}

// ListHooks implements v1.GitHub.
func (_ *stubGitHub) ListHooks(_ context.Context, _ string, _ string) ([]*github.Hook, error) {
	panic("unimplemented")
}

// ListOrganizationRepsitories implements v1.GitHub.
func (*stubGitHub) ListOrganizationRepsitories(context.Context, string) ([]*pb.Repository, error) {
	panic("unimplemented")
}

// ListPackagesByRepository implements v1.GitHub.
func (*stubGitHub) ListPackagesByRepository(context.Context, bool, string, string, int64, int, int) ([]*github.Package, error) {
	panic("unimplemented")
}

// ListPullRequests implements v1.GitHub.
func (*stubGitHub) ListPullRequests(context.Context, string, string, *github.PullRequestListOptions) ([]*github.PullRequest, error) {
	panic("unimplemented")
}

// ListReviews implements v1.GitHub.
func (*stubGitHub) ListReviews(context.Context, string, string, int, *github.ListOptions) ([]*github.PullRequestReview, error) {
	panic("unimplemented")
}

// ListUserRepositories implements v1.GitHub.
func (*stubGitHub) ListUserRepositories(context.Context, string) ([]*pb.Repository, error) {
	panic("unimplemented")
}

// NewRequest implements v1.GitHub.
func (*stubGitHub) NewRequest(string, string, any) (*http.Request, error) {
	panic("unimplemented")
}

// SetCommitStatus implements v1.GitHub.
func (*stubGitHub) SetCommitStatus(context.Context, string, string, string, *github.RepoStatus) (*github.RepoStatus, error) {
	panic("unimplemented")
}

// UpdateBranchProtection implements v1.GitHub.
func (*stubGitHub) UpdateBranchProtection(context.Context, string, string, string, *github.ProtectionRequest) error {
	panic("unimplemented")
}

// GetUserId implements v1.GitHub.
func (*stubGitHub) GetUserId(context.Context) (int64, error) {
	panic("unimplemented")
}

// GetUsername implements v1.GitHub.
func (*stubGitHub) GetUsername(context.Context) (string, error) {
	panic("unimplemented")
}

// GetPrimaryEmail implements v1.GitHub.
func (*stubGitHub) GetPrimaryEmail(context.Context) (string, error) {
	panic("unimplemented")
}

func (*stubGitHub) Clone(context.Context, string, string) (*git.Repository, error) {
	panic("unimplemented")
}
