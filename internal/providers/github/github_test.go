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

package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v56/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/ratecache"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewRestClient(t *testing.T) {
	t.Parallel()

	client, err := NewRestClient(&minderv1.GitHubProviderConfig{
		Endpoint: "https://api.github.com",
	},
		provtelemetry.NewNoopMetrics(),
		nil, "token", "")

	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestArtifactAPIEscapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		testHandler http.HandlerFunc
		cliFn       func(cli *RestClient)
		wantErr     bool
	}{
		{
			name: "GetPackageByName escapes the package name",
			testHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/stacklok/packages/container/helm%2Fmediator", r.URL.RequestURI())
				w.WriteHeader(http.StatusOK)
			},
			cliFn: func(cli *RestClient) {
				_, err := cli.GetPackageByName(context.Background(), true, "stacklok", "container", "helm/mediator")
				assert.NoError(t, err)
			},
		},
		{
			name: "GetPackageVersions escapes the package name",
			testHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/stacklok/packages/container/helm%2Fmediator/versions?package_type=container&page=1&per_page=100&state=active", r.URL.RequestURI())
				w.WriteHeader(http.StatusOK)
			},
			cliFn: func(cli *RestClient) {
				_, err := cli.GetPackageVersions(context.Background(), true, "stacklok", "container", "helm/mediator")
				assert.NoError(t, err)
			},
		},
		{
			name: "GetPackageVersionByTag escapes the package name",
			testHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/stacklok/packages/container/helm%2Fmediator/versions?package_type=container&page=1&per_page=100&state=active", r.URL.RequestURI())
				w.WriteHeader(http.StatusOK)
			},
			cliFn: func(cli *RestClient) {
				_, err := cli.GetPackageVersionByTag(context.Background(), true, "stacklok", "container", "helm/mediator", "v1.0.0")
				assert.NoError(t, err)
			},
		},
		{
			name: "GetPackageVersionByID escapes the package name",
			testHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/stacklok/packages/container/helm%2Fmediator/versions/123", r.URL.RequestURI())
				w.WriteHeader(http.StatusOK)
			},
			cliFn: func(cli *RestClient) {
				_, err := cli.GetPackageVersionById(context.Background(), true, "stacklok", "container", "helm/mediator", 123)
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

			client, err := NewRestClient(&minderv1.GitHubProviderConfig{
				Endpoint: testServer.URL + "/",
			},
				provtelemetry.NewNoopMetrics(),
				nil, "token", "")
			assert.NoError(t, err)
			assert.NotNil(t, client)

			tt.cliFn(client)
		})
	}

}

func TestWaitForRateLimitReset(t *testing.T) {
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

	client, err := NewRestClient(&minderv1.GitHubProviderConfig{Endpoint: server.URL + "/"}, provtelemetry.NewNoopMetrics(), ratecache.NewRestClientCache(context.Background()), token, "mockOwner")
	require.NoError(t, err)

	err = client.CreateComment(context.Background(), "mockOwner", "mockRepo", 1, "Test Comment")
	require.NoError(t, err)

	// Ensure that the second request was made after the rate limit reset
	expectedReq := 2
	assert.Equal(t, expectedReq, reqCount)
}

func TestConcurrentWaitForRateLimitReset(t *testing.T) {
	t.Parallel()

	restClientCache := ratecache.NewRestClientCache(context.Background())
	token := "mockToken-3"
	owner := "mockOwner-3"

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

	wg := sync.WaitGroup{}

	wg.Add(1)
	// Start a goroutine that will make a request to the server, rate limiting the gh client
	go func() {
		defer wg.Done()
		client, err := NewRestClient(&minderv1.GitHubProviderConfig{Endpoint: server.URL + "/"}, provtelemetry.NewNoopMetrics(), restClientCache, token, owner)
		require.NoError(t, err)

		err = client.CreateComment(context.Background(), owner, "mockRepo", 1, "Test Comment")
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

			client, ok := restClientCache.Get(owner, token, db.ProviderTraitGithub)
			require.True(t, ok)

			ghClient, ok := client.(*RestClient)
			require.True(t, ok)

			err := ghClient.CreateComment(ctx, owner, "mockRepo", 1, "Test Comment")
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
