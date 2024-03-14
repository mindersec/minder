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
	"net/http"
	"reflect"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v56/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/config/server"
	mockcrypto "github.com/stacklok/minder/internal/crypto/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	ghprovider "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/ratecache"
	ghrepo "github.com/stacklok/minder/internal/repositories/github"
	mockghrepo "github.com/stacklok/minder/internal/repositories/github/mock"
	mockghhook "github.com/stacklok/minder/internal/repositories/github/webhooks/mock"
	"github.com/stacklok/minder/internal/util/ptr"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestServer_RegisterRepository(t *testing.T) {
	t.Parallel()

	accessToken := "AUTH"

	tests := []struct {
		name    string
		req     *pb.RegisterRepositoryRequest
		repo    github.Repository
		repoErr error
		want    *pb.RegisterRepositoryResponse
		events  []message.Message
		wantErr bool
	}{{
		name: "register repository, invalid upstream ID",
		req: &pb.RegisterRepositoryRequest{
			Provider: "github",
			Repository: &pb.UpstreamRepositoryRef{
				Owner:  "test",
				Name:   "a-test",
				RepoId: 31337,
			},
			Context: &pb.Context{
				Provider: proto.String("github"),
				Project:  proto.String(uuid.NewString()),
			},
		},
		repo: github.Repository{
			Owner: &github.User{
				Login: github.String("test"),
			},
			Name: github.String("a-test"),
			ID:   github.Int64(1234), // NOTE: does not match RPC!
		},
		want: &pb.RegisterRepositoryResponse{
			Result: &pb.RegisterRepoResult{
				Repository: &pb.Repository{
					Id:       proto.String(uuid.NewString()),
					Owner:    "test",
					Name:     "a-test",
					RepoId:   1234,
					HookId:   1,
					HookUuid: uuid.New().String(),
				},
				Status: &pb.RegisterRepoResult_Status{
					Success: true,
				},
			},
		},
		events: []message.Message{{
			Metadata: map[string]string{
				"provider": "github",
			},
			Payload: []byte(`{"repository":1234}`),
		}},
	},
		{
			name: "repo not found",
			req: &pb.RegisterRepositoryRequest{
				Provider: "github",
				Repository: &pb.UpstreamRepositoryRef{
					Owner: "test",
					Name:  "b-test",
				},
				Context: &pb.Context{
					Provider: proto.String("github"),
					Project:  proto.String(uuid.NewString()),
				},
			},
			repoErr: errors.New("Repo not found"),
			want: &pb.RegisterRepositoryResponse{
				Result: &pb.RegisterRepoResult{
					// NOTE: the client as of v0.0.31 expects that the Repository
					// field is always non-null, even if the repo doesn't exist.
					Repository: &pb.Repository{
						Owner: "test",
						Name:  "b-test",
					},
					Status: &pb.RegisterRepoResult_Status{
						Error: proto.String("Repo not found"),
					},
				},
			},
		}}
	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mockdb.NewMockStore(ctrl)

			projectUUID := uuid.MustParse(tt.req.GetContext().GetProject())

			stubClient := StubGitHub{
				ExpectedOwner: tt.req.Repository.GetOwner(),
				ExpectedRepo:  tt.req.Repository.GetName(),
				T:             t,
				Repo:          &tt.repo,
				RepoErr:       tt.repoErr,
			}

			mockStore.EXPECT().
				GetParentProjects(gomock.Any(), projectUUID).
				Return([]uuid.UUID{projectUUID}, nil)
			mockStore.EXPECT().
				ListProvidersByProjectID(gomock.Any(), []uuid.UUID{projectUUID}).
				Return([]db.Provider{{
					ID:         uuid.New(),
					Name:       "github",
					Implements: []db.ProviderType{db.ProviderTypeGithub},
					Version:    provinfv1.V1,
				}}, nil)
			mockStore.EXPECT().
				GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
				Return(db.ProviderAccessToken{
					EncryptedToken: "encryptedToken",
				}, nil).Times(2)
			if tt.repoErr == nil {
				mockStore.EXPECT().
					CreateRepository(gomock.Any(), gomock.Any()).
					Return(db.Repository{
						ID: uuid.MustParse(tt.want.Result.Repository.GetId()),
					}, nil)
			}

			cancelable, cancel := context.WithCancel(context.Background())
			clientCache := ratecache.NewRestClientCache(cancelable)
			defer cancel()
			clientCache.Set("", accessToken, db.ProviderTypeGithub, &stubClient)

			stubEventer := StubEventer{}
			stubWebhookManager := mockghhook.NewMockWebhookManager(ctrl)
			if tt.repoErr == nil {
				stubbedHook := &github.Hook{
					ID: ptr.Ptr[int64](1),
				}
				stubWebhookManager.EXPECT().
					CreateWebhook(gomock.Any(), gomock.Any(), gomock.Eq(tt.req.Repository.Owner), gomock.Eq(tt.req.Repository.Name)).
					Return(tt.want.Result.Repository.HookUuid, stubbedHook, nil)
			}

			mockCryptoEngine := mockcrypto.NewMockEngine(ctrl)
			mockCryptoEngine.EXPECT().DecryptOAuthToken(gomock.Any()).Return(oauth2.Token{AccessToken: accessToken}, nil).AnyTimes()

			s := &Server{
				store:           mockStore,
				cryptoEngine:    mockCryptoEngine,
				restClientCache: clientCache,
				cfg:             &server.Config{},
				evt:             &stubEventer,
				webhookManager:  stubWebhookManager,
			}
			ctx := engine.WithEntityContext(context.Background(), &engine.EntityContext{
				Provider: engine.Provider{Name: tt.req.Context.GetProvider()},
				Project:  engine.Project{ID: projectUUID},
			})

			got, err := s.RegisterRepository(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Server.RegisterRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(tt.events) != len(stubEventer.Sent) {
				t.Fatalf("expected %d events, got %d", len(tt.events), len(stubEventer.Sent))
			}
			for i := range tt.events {
				got := stubEventer.Sent[i]
				want := &tt.events[i]
				if !reflect.DeepEqual(got.Metadata, want.Metadata) {
					t.Errorf("event %d.Metadata = %+v, want %+v", i, got.Metadata, want.Metadata)
				}
				var gotPayload, wantPayload EventPayload
				if err := json.Unmarshal(got.Payload, &gotPayload); err != nil {
					t.Fatalf("failed to unmarshal event %d.Payload: %v", i, err)
				}
				if err := json.Unmarshal(want.Payload, &wantPayload); err != nil {
					t.Fatalf("failed to unmarshal event %d.Payload: %v", i, err)
				}
				if !reflect.DeepEqual(gotPayload.Repository, wantPayload.Repository) {
					t.Errorf("event %d.Payload = %q, want %q", i, string(got.Payload), string(want.Payload))
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Server.RegisterRepository() = %v, want %v", got, tt.want)
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
	repoOwnerAndName = "acme-corp/api-gateway"
	repoID           = "3eb6d254-4163-460f-89f7-44e2ae916e71"
	accessToken      = "TOKEN"
)

var (
	projectID  = uuid.New()
	errDefault = errors.New("oh no")
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
	clientCache.Set("", accessToken, db.ProviderTypeGithub, &StubGitHub{})

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
		repos:           svc,
		cryptoEngine:    mockCryptoEngine,
		restClientCache: clientCache,
	}
}

// TODO: get rid of the need to stub out the entire GitHub client interface
type StubGitHub struct {
	ExpectedOwner string
	ExpectedRepo  string
	T             *testing.T
	Repo          *github.Repository
	RepoErr       error
	UserId        int64
	ExistingHooks []*github.Hook
	DeletedHooks  []int64
	NewHooks      []*github.Hook
}

// FakeGitHub implements v1.GitHub.  Holy wide interface, Batman!
var _ provinfv1.GitHub = (*StubGitHub)(nil)

// CloseSecurityAdvisory implements v1.GitHub.
func (*StubGitHub) CloseSecurityAdvisory(context.Context, string, string, string) error {
	panic("unimplemented")
}

// CreateHook implements v1.GitHub.
func (_ *StubGitHub) CreateHook(_ context.Context, _ string, _ string, _ *github.Hook) (*github.Hook, error) {
	panic("unimplemented")
}

// CreatePullRequest implements v1.GitHub.
func (*StubGitHub) CreatePullRequest(context.Context, string, string, string, string, string, string) (*github.PullRequest, error) {
	panic("unimplemented")
}

// CreateReview implements v1.GitHub.
func (*StubGitHub) CreateReview(context.Context, string, string, int, *github.PullRequestReviewRequest) (*github.PullRequestReview, error) {
	panic("unimplemented")
}

// CreateSecurityAdvisory implements v1.GitHub.
func (*StubGitHub) CreateSecurityAdvisory(context.Context, string, string, string, string, string, []*github.AdvisoryVulnerability) (string, error) {
	panic("unimplemented")
}

// DeleteHook implements v1.GitHub.
func (_ *StubGitHub) DeleteHook(_ context.Context, _ string, _ string, _ int64) (*github.Response, error) {
	panic("unimplemented")
}

// DismissReview implements v1.GitHub.
func (*StubGitHub) DismissReview(context.Context, string, string, int, int64, *github.PullRequestReviewDismissalRequest) (*github.PullRequestReview, error) {
	panic("unimplemented")
}

// Do implements v1.GitHub.
func (*StubGitHub) Do(context.Context, *http.Request) (*http.Response, error) {
	panic("unimplemented")
}

// GetBaseURL implements v1.GitHub.
func (*StubGitHub) GetBaseURL() string {
	panic("unimplemented")
}

// GetBranchProtection implements v1.GitHub.
func (*StubGitHub) GetBranchProtection(context.Context, string, string, string) (*github.Protection, error) {
	panic("unimplemented")
}

// GetOwner implements v1.GitHub.
func (*StubGitHub) GetOwner() string {
	panic("unimplemented")
}

// GetPackageByName implements v1.GitHub.
func (*StubGitHub) GetPackageByName(context.Context, bool, string, string, string) (*github.Package, error) {
	panic("unimplemented")
}

// GetPackageVersionById implements v1.GitHub.
func (*StubGitHub) GetPackageVersionById(context.Context, bool, string, string, string, int64) (*github.PackageVersion, error) {
	panic("unimplemented")
}

// GetPackageVersionByTag implements v1.GitHub.
func (*StubGitHub) GetPackageVersionByTag(context.Context, bool, string, string, string, string) (*github.PackageVersion, error) {
	panic("unimplemented")
}

// GetPackageVersions implements v1.GitHub.
func (*StubGitHub) GetPackageVersions(context.Context, bool, string, string, string) ([]*github.PackageVersion, error) {
	panic("unimplemented")
}

// GetPullRequest implements v1.GitHub.
func (*StubGitHub) GetPullRequest(context.Context, string, string, int) (*github.PullRequest, error) {
	panic("unimplemented")
}

func (*StubGitHub) CreateIssueComment(context.Context, string, string, int, string) (*github.IssueComment, error) {
	panic("unimplemented")
}

func (*StubGitHub) ListIssueComments(context.Context, string, string, int, *github.IssueListCommentsOptions) ([]*github.IssueComment, error) {
	panic("unimplemented")
}

func (*StubGitHub) UpdateIssueComment(context.Context, string, string, int64, string) error {
	panic("unimplemented")
}

func (*StubGitHub) UpdateReview(context.Context, string, string, int, int64, string) (*github.PullRequestReview, error) {
	panic("unimplemented")
}

// GetRepository implements v1.GitHub.
func (s *StubGitHub) GetRepository(_ context.Context, owner string, repo string) (*github.Repository, error) {
	if owner != s.ExpectedOwner {
		s.T.Errorf("expected owner %q, got %q", s.ExpectedOwner, owner)
	}
	if repo != s.ExpectedRepo {
		s.T.Errorf("expected repo %q, got %q", s.ExpectedRepo, repo)
	}
	if s.RepoErr != nil {
		return nil, s.RepoErr
	}
	return s.Repo, nil
}

// GetCredential implements v1.GitHub.
func (*StubGitHub) GetCredential() provinfv1.GitHubCredential {
	panic("unimplemented")
}

// ListAllPackages implements v1.GitHub.
func (*StubGitHub) ListAllPackages(context.Context, bool, string, string, int, int) ([]*github.Package, error) {
	panic("unimplemented")
}

// ListAllRepositories implements v1.GitHub.
func (*StubGitHub) ListAllRepositories(context.Context, bool, string) ([]*github.Repository, error) {
	panic("unimplemented")
}

// ListFiles implements v1.GitHub.
func (*StubGitHub) ListFiles(context.Context, string, string, int, int, int) ([]*github.CommitFile, *github.Response, error) {
	panic("unimplemented")
}

// ListHooks implements v1.GitHub.
func (_ *StubGitHub) ListHooks(_ context.Context, _ string, _ string) ([]*github.Hook, error) {
	panic("unimplemented")
}

// ListOrganizationRepsitories implements v1.GitHub.
func (*StubGitHub) ListOrganizationRepsitories(context.Context, string) ([]*pb.Repository, error) {
	panic("unimplemented")
}

// ListPackagesByRepository implements v1.GitHub.
func (*StubGitHub) ListPackagesByRepository(context.Context, bool, string, string, int64, int, int) ([]*github.Package, error) {
	panic("unimplemented")
}

// ListPullRequests implements v1.GitHub.
func (*StubGitHub) ListPullRequests(context.Context, string, string, *github.PullRequestListOptions) ([]*github.PullRequest, error) {
	panic("unimplemented")
}

// ListReviews implements v1.GitHub.
func (*StubGitHub) ListReviews(context.Context, string, string, int, *github.ListOptions) ([]*github.PullRequestReview, error) {
	panic("unimplemented")
}

// ListUserRepositories implements v1.GitHub.
func (*StubGitHub) ListUserRepositories(context.Context, string) ([]*pb.Repository, error) {
	panic("unimplemented")
}

// NewRequest implements v1.GitHub.
func (*StubGitHub) NewRequest(string, string, any) (*http.Request, error) {
	panic("unimplemented")
}

// SetCommitStatus implements v1.GitHub.
func (*StubGitHub) SetCommitStatus(context.Context, string, string, string, *github.RepoStatus) (*github.RepoStatus, error) {
	panic("unimplemented")
}

// UpdateBranchProtection implements v1.GitHub.
func (*StubGitHub) UpdateBranchProtection(context.Context, string, string, string, *github.ProtectionRequest) error {
	panic("unimplemented")
}

// GetUserId implements v1.GitHub.
func (s *StubGitHub) GetUserId(context.Context) (int64, error) {
	return s.UserId, nil
}

// GetUsername implements v1.GitHub.
func (*StubGitHub) GetUsername(context.Context) (string, error) {
	panic("unimplemented")
}

// GetPrimaryEmail implements v1.GitHub.
func (*StubGitHub) GetPrimaryEmail(context.Context) (string, error) {
	panic("unimplemented")
}

func (*StubGitHub) Clone(context.Context, string, string) (*git.Repository, error) {
	panic("unimplemented")
}

func (*StubGitHub) AddAuthToPushOptions(context.Context, *git.PushOptions) error {
	panic("unimplemented")

}
