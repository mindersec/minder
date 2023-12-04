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
	"testing"

	"github.com/stretchr/testify/assert"

	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewRestClient(t *testing.T) {
	t.Parallel()

	client, err := NewRestClient(context.Background(), &minderv1.GitHubProviderConfig{
		Endpoint: "https://api.github.com",
	},
		provtelemetry.NewNoopMetrics(),
		"token", "")

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

			client, err := NewRestClient(context.Background(), &minderv1.GitHubProviderConfig{
				Endpoint: testServer.URL + "/",
			},
				provtelemetry.NewNoopMetrics(),
				"token", "")
			assert.NoError(t, err)
			assert.NotNil(t, client)

			tt.cliFn(client)
		})
	}

}
