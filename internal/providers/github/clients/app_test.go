// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	config "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/credentials"
	github2 "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/ratecache"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestParseV1AppConfig(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name       string
		config     json.RawMessage
		error      string
		ghEvalFn   func(*testing.T, *minderv1.GitHubAppProviderConfig)
		provEvalFn func(*testing.T, *minderv1.ProviderConfig)
	}{
		{
			name:   "valid app config",
			config: json.RawMessage(`{ "github-app": { "endpoint": "https://api.github.com" } }`),
			ghEvalFn: func(t *testing.T, ghConfig *minderv1.GitHubAppProviderConfig) {
				t.Helper()
				assert.Equal(t, "https://api.github.com", ghConfig.Endpoint)
			},
			provEvalFn: func(t *testing.T, providerConfig *minderv1.ProviderConfig) {
				t.Helper()
				assert.Nil(t, providerConfig)
			},
		},
		{
			name:   "valid app and provider config",
			config: json.RawMessage(`{ "auto_registration": { "entities": { "repository": {"enabled": true} } }, "github-app": { "endpoint": "https://api.github.com" } }`),
			ghEvalFn: func(t *testing.T, ghConfig *minderv1.GitHubAppProviderConfig) {
				t.Helper()
				assert.Equal(t, "https://api.github.com", ghConfig.Endpoint)
			},
			provEvalFn: func(t *testing.T, providerConfig *minderv1.ProviderConfig) {
				t.Helper()
				entityConfig := providerConfig.AutoRegistration.GetEntities()
				assert.NotNil(t, entityConfig)
				assert.Len(t, entityConfig, 1)
				assert.True(t, entityConfig["repository"].Enabled)
			},
		},
		{
			name:   "auto_registration does not validate the enabled entities",
			config: json.RawMessage(`{ "auto_registration": { "entities": { "blah": {"enabled": true} } }, "github-app": { "endpoint": "https://api.github.com" } }`),
			ghEvalFn: func(t *testing.T, ghConfig *minderv1.GitHubAppProviderConfig) {
				t.Helper()
				assert.Nil(t, ghConfig)
			},
			provEvalFn: func(t *testing.T, providerConfig *minderv1.ProviderConfig) {
				t.Helper()
				assert.Nil(t, providerConfig)
			},
			error: "error validating provider config: auto_registration: invalid entity type: blah",
		},
		{
			name:   "missing required github key",
			config: json.RawMessage(`{ "auto_registration": { "entities": { "blah": {"enabled": true} } } }`),
			ghEvalFn: func(t *testing.T, ghConfig *minderv1.GitHubAppProviderConfig) {
				t.Helper()
				assert.Nil(t, ghConfig)
			},
			provEvalFn: func(t *testing.T, providerConfig *minderv1.ProviderConfig) {
				t.Helper()
				assert.Nil(t, providerConfig)
			},
			error: "Field validation for 'GitHubApp' failed on the 'required' tag",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			providerConfig, gitHubConfig, err := ParseV1AppConfig(scenario.config)
			if scenario.error != "" {
				assert.Error(t, err)
				assert.Nil(t, providerConfig)
				assert.Contains(t, err.Error(), scenario.error)
			} else {
				assert.NoError(t, err)
				scenario.provEvalFn(t, providerConfig)
				scenario.ghEvalFn(t, gitHubConfig)
			}
		})
	}
}

func TestNewGitHubAppProvider(t *testing.T) {
	t.Parallel()

	client, err := NewGitHubAppProvider(
		&minderv1.GitHubAppProviderConfig{Endpoint: "https://api.github.com"},
		&config.GitHubAppConfig{},
		nil,
		credentials.NewGitHubTokenCredential("token"),
		github.NewClient(http.DefaultClient),
		NewGitHubClientFactory(provtelemetry.NewNoopMetrics()),
		false,
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestUserInfo(t *testing.T) {
	t.Parallel()
	appName := "test-app"
	appId := int64(123456)
	expectedUserId := int64(123456789)

	ctx := context.Background()

	client, err := NewGitHubAppProvider(
		&minderv1.GitHubAppProviderConfig{Endpoint: "https://api.github.com"},
		&config.GitHubAppConfig{
			AppName: appName,
			AppID:   appId,
			UserID:  expectedUserId,
		},
		ratecache.NewRestClientCache(context.Background()),
		credentials.NewGitHubTokenCredential("token"),
		github.NewClient(http.DefaultClient),
		NewGitHubClientFactory(provtelemetry.NewNoopMetrics()),
		false,
	)
	assert.NoError(t, err)

	userId, err := client.GetUserId(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedUserId, userId)

	name, err := client.GetName(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test-app[bot]", name)

	login, err := client.GetLogin(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test-app[bot]", login)

	email, err := client.GetPrimaryEmail(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "123456789+github-actions[bot]@users.noreply.github.com", email)
}

func TestArtifactAPIEscapesApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		testHandler http.HandlerFunc
		cliFn       func(cli *github2.GitHub)
		wantErr     bool
	}{
		{
			name: "GetPackageByName escapes the package name",
			testHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/stacklok/packages/container/helm%2Fmediator", r.URL.RequestURI())
				w.WriteHeader(http.StatusOK)
			},
			cliFn: func(cli *github2.GitHub) {
				_, err := cli.GetPackageByName(context.Background(), "stacklok", "container", "helm/mediator")
				assert.NoError(t, err)
			},
		},
		{
			name: "GetPackageVersionByID escapes the package name",
			testHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/stacklok/packages/container/helm%2Fmediator/versions/123", r.URL.RequestURI())
				w.WriteHeader(http.StatusOK)
			},
			cliFn: func(cli *github2.GitHub) {
				_, err := cli.GetPackageVersionById(context.Background(), "stacklok", "container", "helm/mediator", 123)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testServer := httptest.NewServer(tt.testHandler)
			defer testServer.Close()

			packageListingClient := github.NewClient(http.DefaultClient)
			testServerUrl, err := url.Parse(testServer.URL + "/")
			assert.NoError(t, err)
			packageListingClient.BaseURL = testServerUrl

			client, err := NewGitHubAppProvider(
				&minderv1.GitHubAppProviderConfig{Endpoint: testServer.URL + "/"},
				&config.GitHubAppConfig{},
				ratecache.NewRestClientCache(context.Background()),
				credentials.NewGitHubTokenCredential("token"),
				packageListingClient,
				NewGitHubClientFactory(provtelemetry.NewNoopMetrics()),
				true,
			)

			assert.NoError(t, err)
			assert.NotNil(t, client)

			tt.cliFn(client)
		})
	}

}

func TestListPackagesByRepository(t *testing.T) {
	t.Parallel()

	accessToken := "token"
	repositoryId := int64(1234)
	gitHubPackage := github.Package{
		Name: github.String("test-package"),
		Repository: &github.Repository{
			ID: github.Int64(repositoryId),
		},
	}

	tests := []struct {
		name          string
		testHandler   http.HandlerFunc
		ExpectedError string
	}{
		{
			name: "ListPackagesByRepository returns matching package",
			testHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/owner/packages?package_type=repo&page=1&per_page=1", r.URL.RequestURI())
				data := []github.Package{gitHubPackage}
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "ListPackagesByRepository filters out non-matching packages",
			testHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/owner/packages?package_type=repo&page=1&per_page=1", r.URL.RequestURI())
				packageOtherRepo := github.Package{
					Name: github.String("other-package"),
					Repository: &github.Repository{
						ID: github.Int64(5678),
					},
				}
				data := []github.Package{gitHubPackage, packageOtherRepo}
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testServer := httptest.NewServer(tt.testHandler)
			defer testServer.Close()

			packageListingClient := github.NewClient(http.DefaultClient)
			testServerUrl, err := url.Parse(testServer.URL + "/")
			assert.NoError(t, err)
			packageListingClient.BaseURL = testServerUrl

			provider, err := NewGitHubAppProvider(
				&minderv1.GitHubAppProviderConfig{},
				&config.GitHubAppConfig{
					FallbackToken: accessToken,
				},
				ratecache.NewRestClientCache(context.Background()),
				credentials.NewGitHubTokenCredential(accessToken),
				packageListingClient,
				NewGitHubClientFactory(provtelemetry.NewNoopMetrics()),
				true,
			)
			assert.NoError(t, err)
			assert.NotNil(t, provider)

			packages, err := provider.ListPackagesByRepository(context.Background(), "owner", "repo", repositoryId, 1, 1)
			if tt.ExpectedError == "" {
				assert.NoError(t, err)
				assert.Len(t, packages, 1)
				assert.Equal(t, gitHubPackage.Name, packages[0].Name)
			} else {
				assert.Error(t, err)
			}

		})
	}
}

func TestWaitForRateLimitResetApp(t *testing.T) {
	t.Parallel()

	token := "mockToken-2"

	reqCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reqCount++
		if reqCount == 1 {
			w.Header().Set("x-ratelimit-remaining", "0")
			w.Header().Set("x-ratelimit-reset", strconv.FormatInt(time.Now().Add(1*time.Second).Unix(), 10))
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	packageListingClient := github.NewClient(http.DefaultClient)
	testServerUrl, err := url.Parse(server.URL + "/")
	assert.NoError(t, err)
	packageListingClient.BaseURL = testServerUrl

	client, err := NewGitHubAppProvider(
		&minderv1.GitHubAppProviderConfig{Endpoint: server.URL + "/"},
		&config.GitHubAppConfig{},
		ratecache.NewRestClientCache(context.Background()),
		credentials.NewGitHubTokenCredential(token),
		packageListingClient,
		NewGitHubClientFactory(provtelemetry.NewNoopMetrics()),
		false,
	)
	require.NoError(t, err)

	_, err = client.CreateIssueComment(context.Background(), "mockOwner", "mockRepo", 1, "Test Comment")
	require.NoError(t, err)

	// Ensure that the second request was made after the rate limit reset
	expectedReq := 2
	assert.Equal(t, expectedReq, reqCount)
}

func TestConcurrentWaitForRateLimitResetApp(t *testing.T) {
	t.Parallel()

	restClientCache := ratecache.NewRestClientCache(context.Background())
	token := "mockToken-3"
	owner := ""

	var reqCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		reqCount++
		defer mu.Unlock()

		if reqCount == 1 {
			w.Header().Set("x-ratelimit-remaining", "0")
			// 50 minute reset time is more than max allowed wait time
			rateLimitResetTime := 50 * time.Minute
			w.Header().Set("x-ratelimit-reset", strconv.FormatInt(time.Now().Add(rateLimitResetTime).Unix(), 10))
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	packageListingClient := github.NewClient(http.DefaultClient)
	testServerUrl, err := url.Parse(server.URL + "/")
	assert.NoError(t, err)
	packageListingClient.BaseURL = testServerUrl

	wg := sync.WaitGroup{}

	wg.Add(1)
	// Start a goroutine that will make a request to the server, rate limiting the gh client
	go func() {
		defer wg.Done()
		client, err := NewGitHubAppProvider(
			&minderv1.GitHubAppProviderConfig{Endpoint: server.URL + "/"},
			&config.GitHubAppConfig{},
			restClientCache,
			credentials.NewGitHubTokenCredential(token),
			packageListingClient,
			NewGitHubClientFactory(provtelemetry.NewNoopMetrics()),
			false,
		)
		require.NoError(t, err)

		_, err = client.CreateIssueComment(context.Background(), owner, "mockRepo", 1, "Test Comment")
		var rateLimitErr *github.RateLimitError
		require.ErrorAs(t, err, &rateLimitErr)
	}()

	wg.Wait()

	ctx := context.Background()

	numGoroutines := 5
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			client, ok := restClientCache.Get(owner, token, db.ProviderTypeGithub)
			require.True(t, ok)

			ghClient, ok := client.(*github2.GitHub)
			require.True(t, ok)

			_, err := ghClient.CreateIssueComment(ctx, owner, "mockRepo", 1, "Test Comment")
			var rateLimitErr *github.RateLimitError
			require.ErrorAs(t, err, &rateLimitErr)
		}()
	}

	wg.Wait()

	// We only want to see one request to the server, the rest should be rate limited and gh client
	// should return a made up response i.e. not a real http response by contacting the server
	expectedServerReq := 1
	assert.Equal(t, expectedServerReq, reqCount)
}
