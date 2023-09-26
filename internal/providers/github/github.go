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

// Package github provides a client for interacting with the GitHub API
package github

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"

	"github.com/stacklok/mediator/internal/db"
	provifv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

const (
	// ExpensiveRestCallTimeout is the timeout for expensive REST calls
	ExpensiveRestCallTimeout = 15 * time.Second
)

// Github is the string that represents the GitHub provider
const Github = "github"

// Implements is the list of provider types that the GitHub provider implements
var Implements = []db.ProviderType{
	db.ProviderTypeGithub,
	db.ProviderTypeGit,
	db.ProviderTypeRest,
}

// RestClient is the struct that contains the GitHub REST API client
type RestClient struct {
	client *github.Client
	token  string
	owner  string
}

// Ensure that the GitHub client implements the GitHub interface
var _ provifv1.GitHub = (*RestClient)(nil)

// NewRestClient creates a new GitHub REST API client
// BaseURL defaults to the public GitHub API, if needing to use a customer domain
// endpoint (as is the case with GitHub Enterprise), set the Endpoint field in
// the GitHubConfig struct
func NewRestClient(ctx context.Context, config *provifv1.GitHubConfig, token string, owner string) (*RestClient, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	ghClient := github.NewClient(tc)

	if config.Endpoint != "" {
		parsedURL, err := url.Parse(config.Endpoint)
		if err != nil {
			return nil, err
		}
		ghClient.BaseURL = parsedURL
	}

	return &RestClient{
		client: ghClient,
		token:  token,
		owner:  owner,
	}, nil
}

// ParseV1Config parses the raw config into a GitHubConfig struct
func ParseV1Config(rawCfg json.RawMessage) (*provifv1.GitHubConfig, error) {
	type wrapper struct {
		GitHub *provifv1.GitHubConfig `json:"github" yaml:"github" mapstructure:"github" validate:"required"`
	}

	var w wrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	return w.GitHub, nil
}
