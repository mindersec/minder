package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v63/github"
	"github.com/mindersec/minder/internal/db"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/timestamppb"

	mock_github "github.com/mindersec/minder/internal/providers/github/mock"
	mock_ratecache "github.com/mindersec/minder/internal/providers/ratecache/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	config "github.com/mindersec/minder/pkg/config/server"
)

type testGitHub struct {
	gh       *GitHub
	delegate *mock_github.MockDelegate
	cache    *mock_ratecache.MockRestClientCache
	client   *github.Client
}

func setupTest(t *testing.T) *testGitHub {
	ctrl := gomock.NewController(t)
	delegate := mock_github.NewMockDelegate(ctrl)
	cache := mock_ratecache.NewMockRestClientCache(ctrl)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "dummy-token"})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	gh := &GitHub{
		client:   client,
		cache:    cache,
		delegate: delegate,
	}

	return &testGitHub{
		gh:       gh,
		delegate: delegate,
		cache:    cache,
		client:   client,
	}
}

func TestWaitForRateLimitReset(t *testing.T) {
	tests := []struct {
		name        string
		inputErr    error
		setupMocks  func(*testGitHub)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "primary rate limit error with reset time under max wait",
			inputErr: &github.RateLimitError{
				Rate: github.Rate{
					Remaining: 0,
					Reset:     github.Timestamp{Time: time.Now().Add(100 * time.Millisecond)},
				},
				Response: &http.Response{
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{Path: "/test"},
					},
				},
			},
			setupMocks: func(th *testGitHub) {
				th.delegate.EXPECT().GetCredential().Return(&mockCredential{}).AnyTimes()
				th.delegate.EXPECT().GetOwner().Return("test-owner").AnyTimes()
				th.cache.EXPECT().Set(
					"test-owner",
					gomock.Any(),
					db.ProviderTypeGithub,
					th.gh,
				).Return()
			},
			wantErr: false,
		},
		{
			name: "primary rate limit error with reset time over max wait",
			inputErr: &github.RateLimitError{
				Rate: github.Rate{
					Remaining: 0,
					Reset:     github.Timestamp{Time: time.Now().Add(10 * time.Minute)},
				},
				Response: &http.Response{
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{Path: "/test"},
					},
				},
			},
			setupMocks: func(th *testGitHub) {
				th.delegate.EXPECT().GetCredential().Return(&mockCredential{}).AnyTimes()
				th.delegate.EXPECT().GetOwner().Return("test-owner").AnyTimes()
				th.cache.EXPECT().Set(
					"test-owner",
					gomock.Any(),
					db.ProviderTypeGithub,
					th.gh,
				).Return()
			},
			wantErr:     true,
			expectedErr: &github.RateLimitError{},
		},
		{
			name: "abuse rate limit error with retry after under max wait",
			inputErr: &github.AbuseRateLimitError{
				RetryAfter: func() *time.Duration {
					d := 100 * time.Millisecond
					return &d
				}(),
				Response: &http.Response{
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{Path: "/test"},
					},
				},
			},
			setupMocks: func(th *testGitHub) {
				th.delegate.EXPECT().GetCredential().Return(&mockCredential{}).AnyTimes()
				th.delegate.EXPECT().GetOwner().Return("test-owner").AnyTimes()
				th.cache.EXPECT().Set(
					"test-owner",
					gomock.Any(),
					db.ProviderTypeGithub,
					th.gh,
				).Return()
			},
			wantErr: false,
		},
		{
			name:       "non-rate limit error",
			inputErr:   errors.New("random error"),
			setupMocks: func(th *testGitHub) {},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := setupTest(t)
			tt.setupMocks(th)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := th.gh.waitForRateLimitReset(ctx, tt.inputErr)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.IsType(t, tt.expectedErr, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPerformWithRetry(t *testing.T) {
	tests := []struct {
		name       string
		operation  func() (string, error)
		wantErr    bool
		wantResult string
	}{
		{
			name: "successful operation",
			operation: func() (string, error) {
				return "success", nil
			},
			wantErr:    false,
			wantResult: "success",
		},
		{
			name: "permanent error",
			operation: func() (string, error) {
				return "", errors.New("permanent error")
			},
			wantErr: true,
		},
		{
			name: "rate limit error then success",
			operation: func() (string, error) {
				return "", &github.RateLimitError{
					Rate: github.Rate{
						Remaining: 0,
						Reset:     github.Timestamp{Time: time.Now().Add(1 * time.Second)},
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := performWithRetry(context.Background(), tt.operation)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

type mockCredential struct {
	provifv1.GitHubCredential
}

func (m *mockCredential) GetCacheKey() string {
	return "mock-cache-key"
}

func TestListPackagesByRepository(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		artifactType   string
		repositoryId   int64
		pageNumber     int
		itemsPerPage   int
		setupMocks     func(*testGitHub)
		expectedResult []*github.Package
		expectedError  error
	}{
		{
			name:         "successful listing with single page",
			owner:        "test-owner",
			artifactType: "container",
			repositoryId: 123,
			pageNumber:   1,
			itemsPerPage: 100,
			setupMocks: func(th *testGitHub) {
				ctrl := gomock.NewController(t)
				mockDelegate := mock_github.NewMockDelegate(ctrl)
				mockDelegate.EXPECT().IsOrg().Return(true)
				th.gh.delegate = mockDelegate

				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body: io.NopCloser(strings.NewReader(`[
								{
									"id": 1,
									"name": "package1",
									"repository": {
										"id": 123
									}
								}
							]`)),
							Header: make(http.Header),
						},
					},
				}

				ghClient := github.NewClient(client)
				th.gh.packageListingClient = ghClient
				th.gh.client = ghClient
			},
			expectedResult: []*github.Package{
				{
					ID:         github.Int64(1),
					Name:       github.String("package1"),
					Repository: &github.Repository{ID: github.Int64(123)},
				},
			},
		},
		{
			name:         "no package listing client available",
			owner:        "test-owner",
			artifactType: "container",
			repositoryId: 123,
			pageNumber:   1,
			itemsPerPage: 100,
			setupMocks: func(th *testGitHub) {
				ctrl := gomock.NewController(t)
				mockDelegate := mock_github.NewMockDelegate(ctrl)
				th.gh.delegate = mockDelegate

				th.gh.packageListingClient = nil
				th.gh.client = nil
			},
			expectedError: ErrNoPackageListingClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := setupTest(t)
			tt.setupMocks(th)

			result, err := th.gh.ListPackagesByRepository(
				context.Background(),
				tt.owner,
				tt.artifactType,
				tt.repositoryId,
				tt.pageNumber,
				tt.itemsPerPage,
			)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestGetPackageVersions(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		packageType    string
		packageName    string
		setupMocks     func(*testGitHub)
		expectedResult []*github.PackageVersion
		expectedError  error
	}{
		{
			name:        "successful version listing with single page",
			owner:       "test-owner",
			packageType: "container",
			packageName: "test-package",
			setupMocks: func(th *testGitHub) {
				ctrl := gomock.NewController(t)
				mockDelegate := mock_github.NewMockDelegate(ctrl)
				mockDelegate.EXPECT().IsOrg().Return(true)
				th.gh.delegate = mockDelegate

				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body: io.NopCloser(strings.NewReader(`[
								{
									"id": 1,
									"name": "1.0.0"
								}
							]`)),
							Header: make(http.Header),
						},
					},
				}
				ghClient := github.NewClient(client)
				th.gh.client = ghClient
			},
			expectedResult: []*github.PackageVersion{
				{
					ID:   github.Int64(1),
					Name: github.String("1.0.0"),
				},
			},
		},
		{
			name:        "successful version listing with multiple versions",
			owner:       "test-owner",
			packageType: "container",
			packageName: "test-package",
			setupMocks: func(th *testGitHub) {
				ctrl := gomock.NewController(t)
				mockDelegate := mock_github.NewMockDelegate(ctrl)
				mockDelegate.EXPECT().IsOrg().Return(true)
				th.gh.delegate = mockDelegate

				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body: io.NopCloser(strings.NewReader(`[
								{
									"id": 1,
									"name": "1.0.0"
								},
								{
									"id": 2,
									"name": "1.1.0"
								},
								{
									"id": 3,
									"name": "2.0.0"
								}
							]`)),
							Header: make(http.Header),
						},
					},
				}
				ghClient := github.NewClient(client)
				th.gh.client = ghClient
			},
			expectedResult: []*github.PackageVersion{
				{
					ID:   github.Int64(1),
					Name: github.String("1.0.0"),
				},
				{
					ID:   github.Int64(2),
					Name: github.String("1.1.0"),
				},
				{
					ID:   github.Int64(3),
					Name: github.String("2.0.0"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := setupTest(t)
			tt.setupMocks(th)

			result, err := th.gh.getPackageVersions(
				context.Background(),
				tt.owner,
				tt.packageType,
				tt.packageName,
			)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestGetArtifactVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		owner          string
		artifactType   string
		artifactName   string
		pageNumber     int
		itemsPerPage   int
		setupMocks     func(*testGitHub)
		expectedResult []*minderv1.ArtifactVersion
		expectedError  error
	}{
		{
			name:         "successful fetch",
			owner:        "test-owner",
			artifactType: "container",
			artifactName: "test-package",
			pageNumber:   1,
			itemsPerPage: 100,
			setupMocks: func(th *testGitHub) {
				ctrl := gomock.NewController(t)
				mockDelegate := mock_github.NewMockDelegate(ctrl)
				mockDelegate.EXPECT().IsOrg().Return(true)
				th.gh.delegate = mockDelegate

				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body: io.NopCloser(strings.NewReader(`[
								{
									"id": 123,
									"name": "1",
									"created_at": "2023-01-01T00:00:00Z",
									"metadata": {
										"container": {
											"tags": ["latest"]
										}
									}
								}
							]`)),
							Header: make(http.Header),
						},
					},
				}

				ghClient := github.NewClient(client)
				th.gh.client = ghClient
			},
			expectedResult: []*minderv1.ArtifactVersion{
				{
					VersionId: 0,
					Tags:      []string{"latest"},
					Sha:       "1",
					CreatedAt: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			expectedError: nil,
		},
		{
			name:         "server error",
			owner:        "test-owner",
			artifactType: "container",
			artifactName: "test-package",
			pageNumber:   1,
			itemsPerPage: 100,
			setupMocks: func(th *testGitHub) {
				th.delegate.EXPECT().IsOrg().Return(true).AnyTimes()

				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusUnauthorized,
							Body:       io.NopCloser(strings.NewReader(`{"message": "Bad credentials"}`)),
						},
					},
				}

				ghClient := github.NewClient(client)
				th.gh.packageListingClient = ghClient
			},
			expectedResult: nil,
			expectedError:  errors.New("error retrieving artifact versions: GET https://api.github.com/orgs/test-owner/packages/container/test-package/versions?package_type=container&page=1&per_page=100&state=active: 401 Bad credentials []"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			th := setupTest(t)
			tt.setupMocks(th)

			result, err := th.gh.GetArtifactVersions(
				context.Background(),
				&minderv1.Artifact{
					Owner: tt.owner,
					Type:  tt.artifactType,
					Name:  tt.artifactName,
				},
				&testFilter{
					pageNumber:   int32(tt.pageNumber),
					itemsPerPage: int32(tt.itemsPerPage),
				},
			)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

type testFilter struct {
	pageNumber   int32
	itemsPerPage int32
}

func (f *testFilter) GetPageNumber() int32   { return f.pageNumber }
func (f *testFilter) GetItemsPerPage() int32 { return f.itemsPerPage }
func (f *testFilter) IsSkippable(createdAt time.Time, tags []string) error {
	return nil
}

type mockTransport struct {
	response *http.Response
}

func (t *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return t.response, nil
}

func TestIsMinderHook(t *testing.T) {
	tests := []struct {
		name         string
		hook         *github.Hook
		hostURL      string
		wantIsMinder bool
		wantErr      bool
	}{
		{
			name: "valid minder hook",
			hook: &github.Hook{
				Config: &github.HookConfig{
					URL: github.String("https://minder.example.com/webhook"),
				},
			},
			hostURL:      "minder.example.com",
			wantIsMinder: true,
			wantErr:      false,
		},
		{
			name: "non-minder hook",
			hook: &github.Hook{
				Config: &github.HookConfig{
					URL: github.String("https://other-service.com/webhook"),
				},
			},
			hostURL:      "minder.example.com",
			wantIsMinder: false,
			wantErr:      false,
		},
		{
			name: "empty hook config",
			hook: &github.Hook{
				Config: &github.HookConfig{},
			},
			hostURL:      "minder.example.com",
			wantIsMinder: false,
			wantErr:      true,
		},
		{
			name: "invalid URL in hook config",
			hook: &github.Hook{
				Config: &github.HookConfig{
					URL: github.String("://invalid-url"),
				},
			},
			hostURL:      "minder.example.com",
			wantIsMinder: false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isMinder, err := IsMinderHook(tt.hook, tt.hostURL)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantIsMinder, isMinder)
			}
		})
	}
}

func TestCreateHook(t *testing.T) {
	tests := []struct {
		name        string
		owner       string
		repo        string
		hook        *github.Hook
		setupMocks  func(*testGitHub)
		wantHook    *github.Hook
		wantErr     bool
		expectedErr error
	}{
		{
			name:  "successful hook creation",
			owner: "test-owner",
			repo:  "test-repo",
			hook: &github.Hook{
				Config: &github.HookConfig{
					URL:         github.String("https://minder.example.com/webhook"),
					ContentType: github.String("json"),
				},
				Events: []string{"push", "pull_request"},
				Active: github.Bool(true),
			},
			setupMocks: func(th *testGitHub) {
				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusCreated,
							Body: io.NopCloser(strings.NewReader(`{
								"id": 1,
								"url": "https://api.github.com/repos/test-owner/test-repo/hooks/1",
								"config": {
									"url": "https://minder.example.com/webhook",
									"content_type": "json"
								},
								"events": ["push", "pull_request"],
								"active": true
							}`)),
							Header: make(http.Header),
						},
					},
				}
				ghClient := github.NewClient(client)
				th.gh.client = ghClient
			},
			wantHook: &github.Hook{
				ID:  github.Int64(1),
				URL: github.String("https://api.github.com/repos/test-owner/test-repo/hooks/1"),
				Config: &github.HookConfig{
					URL:         github.String("https://minder.example.com/webhook"),
					ContentType: github.String("json"),
				},
				Events: []string{"push", "pull_request"},
				Active: github.Bool(true),
			},
			wantErr: false,
		},
		{
			name:  "unauthorized error",
			owner: "test-owner",
			repo:  "test-repo",
			hook: &github.Hook{
				Config: &github.HookConfig{
					URL: github.String("https://minder.example.com/webhook"),
				},
			},
			setupMocks: func(th *testGitHub) {
				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusUnauthorized,
							Body:       io.NopCloser(strings.NewReader(`{"message": "Bad credentials"}`)),
							Header:     make(http.Header),
						},
					},
				}
				ghClient := github.NewClient(client)
				th.gh.client = ghClient
			},
			wantErr: true,
			expectedErr: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusUnauthorized,
				},
				Message: "Bad credentials",
			},
		},
		{
			name:  "repository not found",
			owner: "test-owner",
			repo:  "test-repo",
			hook: &github.Hook{
				Config: &github.HookConfig{
					URL: github.String("https://minder.example.com/webhook"),
				},
			},
			setupMocks: func(th *testGitHub) {
				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusNotFound,
							Body:       io.NopCloser(strings.NewReader(`{"message": "Not Found"}`)),
							Header:     make(http.Header),
						},
					},
				}
				ghClient := github.NewClient(client)
				th.gh.client = ghClient
			},
			wantErr: true,
			expectedErr: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusNotFound,
				},
				Message: "Not Found",
			},
		},
		{
			name:  "rate limit exceeded",
			owner: "test-owner",
			repo:  "test-repo",
			hook: &github.Hook{
				Config: &github.HookConfig{
					URL: github.String("https://minder.example.com/webhook"),
				},
			},
			setupMocks: func(th *testGitHub) {
				headers := make(http.Header)
				headers.Set("X-RateLimit-Limit", "60")
				headers.Set("X-RateLimit-Remaining", "0")
				headers.Set("X-RateLimit-Reset", "1632182400")
				headers.Set("Content-Type", "application/json; charset=utf-8")

				client := &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusForbidden,
							Body: io.NopCloser(strings.NewReader(`{
								"message": "API rate limit exceeded for xxx.xxx.xxx.xxx",
								"documentation_url": "https://docs.github.com/rest/overview/resources-in-the-rest-api#rate-limiting"
							}`)),
							Header: headers,
							Request: &http.Request{
								Method: "GET",
								URL: &url.URL{
									Scheme: "https",
									Host:   "api.github.com",
									Path:   "/repos/test-owner/test-repo/hooks",
								},
							},
						},
					},
				}
				ghClient := github.NewClient(client)
				th.gh.client = ghClient

				th.delegate.EXPECT().GetCredential().Return(&mockCredential{}).AnyTimes()
				th.delegate.EXPECT().GetOwner().Return("test-owner").AnyTimes()
				th.cache.EXPECT().Set(
					"test-owner",
					gomock.Any(),
					db.ProviderTypeGithub,
					th.gh,
				).AnyTimes()
			},
			wantErr:     true,
			expectedErr: &github.RateLimitError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := setupTest(t)
			tt.setupMocks(th)

			hook, err := th.gh.CreateHook(context.Background(), tt.owner, tt.repo, tt.hook)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.IsType(t, tt.expectedErr, err)

					var ghErr *github.ErrorResponse
					if errors.As(err, &ghErr) {
						var expectedErr *github.ErrorResponse
						errors.As(tt.expectedErr, &expectedErr)
						assert.Equal(t, expectedErr.Message, ghErr.Message)
						assert.Equal(t, expectedErr.Response.StatusCode, ghErr.Response.StatusCode)
					}

					var rateLimitErr *github.RateLimitError
					if errors.As(err, &rateLimitErr) {
						// For rate limit errors, we just check the type match
						// since the specific rate limit details may vary
						var rateLimitError *github.RateLimitError
						ok := errors.As(tt.expectedErr, &rateLimitError)
						assert.True(t, ok)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantHook, hook)
			}
		})
	}
}

func TestCanHandleOwner(t *testing.T) {
	tests := []struct {
		name     string
		provider db.Provider
		owner    string
		want     bool
	}{
		{
			name: "github app provider with matching owner",
			provider: db.Provider{
				Name:  "github-app-test-owner",
				Class: db.ProviderClassGithubApp,
			},
			owner: "test-owner",
			want:  true,
		},
		{
			name: "github app provider with non-matching owner",
			provider: db.Provider{
				Name:  "github-app-different-owner",
				Class: db.ProviderClassGithubApp,
			},
			owner: "test-owner",
			want:  false,
		},
		{
			name: "github provider class",
			provider: db.Provider{
				Name:  "any-name",
				Class: db.ProviderClassGithub,
			},
			owner: "test-owner",
			want:  true,
		},
		{
			name: "non-github provider class",
			provider: db.Provider{
				Name:  "any-name",
				Class: "other-provider",
			},
			owner: "test-owner",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanHandleOwner(context.Background(), tt.provider, tt.owner)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewFallbackTokenClient(t *testing.T) {
	tests := []struct {
		name      string
		appConfig config.ProviderConfig
		wantNil   bool
	}{
		{
			name: "valid fallback token",
			appConfig: config.ProviderConfig{
				GitHubApp: &config.GitHubAppConfig{
					FallbackToken: "valid-token",
				},
			},
			wantNil: false,
		},
		{
			name: "empty fallback token",
			appConfig: config.ProviderConfig{
				GitHubApp: &config.GitHubAppConfig{
					FallbackToken: "",
				},
			},
			wantNil: true,
		},
		{
			name: "nil github app config",
			appConfig: config.ProviderConfig{
				GitHubApp: nil,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewFallbackTokenClient(tt.appConfig)
			if tt.wantNil {
				assert.Nil(t, client)
			} else {
				assert.NotNil(t, client)
			}
		})
	}
}
