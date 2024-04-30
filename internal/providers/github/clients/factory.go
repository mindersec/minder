// Copyright 2024 Stacklok, Inc
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

// Package clients contains the GitHub client factory
package clients

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	gogithub "github.com/google/go-github/v61/github"
	"github.com/motemen/go-loghttp"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/telemetry"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// GitHubClientFactory creates instances of the GitHub API client
type GitHubClientFactory interface {
	// Build creates an instance of the GitHub Client
	// `baseURL` should be set to the empty string if there is no need to
	// override the default GitHub URL
	Build(baseURL string, credential provifv1.GitHubCredential) (*gogithub.Client, error)
}

type githubClientFactory struct {
	metrics telemetry.HttpClientMetrics
}

// NewGitHubClientFactory creates a new instance of GitHubClientFactory
func NewGitHubClientFactory(metrics telemetry.HttpClientMetrics) GitHubClientFactory {
	return &githubClientFactory{metrics: metrics}
}

func (g *githubClientFactory) Build(baseURL string, credential provifv1.GitHubCredential) (*gogithub.Client, error) {
	tc := &http.Client{
		Transport: &oauth2.Transport{
			Base:   http.DefaultClient.Transport,
			Source: credential.GetAsOAuth2TokenSource(),
		},
	}

	transport, err := g.metrics.NewDurationRoundTripper(tc.Transport, db.ProviderTypeGithub)
	if err != nil {
		return nil, fmt.Errorf("error creating duration round tripper: %w", err)
	}

	// If $MINDER_LOG_GITHUB_REQUESTS is set, wrap the transport in a logger
	// to record all calls and responses to from GitHub:
	if os.Getenv("MINDER_LOG_GITHUB_REQUESTS") != "" {
		transport = &loghttp.Transport{
			Transport: transport,
			LogRequest: func(req *http.Request) {
				zerolog.Ctx(req.Context()).Debug().
					Str("type", "REQ").
					Str("method", req.Method).
					Msg(req.URL.String())
			},
			LogResponse: func(resp *http.Response) {
				zerolog.Ctx(resp.Request.Context()).Debug().
					Str("type", "RESP").
					Str("method", resp.Request.Method).
					Str("status", fmt.Sprintf("%d", resp.StatusCode)).
					Str("rate-limit", fmt.Sprintf("%s/%s",
						resp.Request.Header.Get("x-ratelimit-used"),
						resp.Request.Header.Get("x-ratelimit-remaining"),
					)).
					Msg(resp.Request.URL.String())

			},
		}
	}

	tc.Transport = transport
	ghClient := gogithub.NewClient(tc)

	if baseURL != "" {
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}
		ghClient.BaseURL = parsedURL
	}

	return ghClient, nil
}
