//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package gitlab provides the GitLab OAuth provider implementation
package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/oauth2"

	config "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// Class is the string that represents the GitLab provider class
const Class = "gitlab"

// Implements is the list of provider types that the DockerHub provider implements
var Implements = []db.ProviderType{
	db.ProviderTypeGit,
	db.ProviderTypeRest,
	db.ProviderTypeRepoLister,
}

// AuthorizationFlows is the list of authorization flows that the DockerHub provider supports
var AuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowUserInput,
	db.AuthorizationFlowOauth2AuthorizationCodeFlow,
}

// Ensure that the GitLab provider implements the right interfaces
var _ provifv1.Git = (*gitlabClient)(nil)
var _ provifv1.REST = (*gitlabClient)(nil)
var _ provifv1.RepoLister = (*gitlabClient)(nil)

type gitlabClient struct {
	cred      provifv1.GitLabCredential
	cli       *http.Client
	glcfg     *minderv1.GitLabProviderConfig
	gitConfig config.GitConfig
}

// New creates a new GitLab provider
func New(cred provifv1.GitLabCredential, cfg *minderv1.GitLabProviderConfig) (*gitlabClient, error) {
	// TODO: We need a context here.
	cli := oauth2.NewClient(context.Background(), cred.GetAsOAuth2TokenSource())

	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://gitlab.com/api/v4/"
	}

	return &gitlabClient{
		cred:  cred,
		cli:   cli,
		glcfg: cfg,
		// TODO: Add git config
	}, nil
}

type glConfigWrapper struct {
	GitLab *minderv1.GitLabProviderConfig `json:"gitlab" yaml:"gitlab" mapstructure:"gitlab" validate:"required"`
}

// ParseV1Config parses the raw configuration into a GitLabProviderConfig
//
// TODO: This should be moved to a common location
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.GitLabProviderConfig, error) {
	var cfg glConfigWrapper
	if err := json.Unmarshal(rawCfg, &cfg); err != nil {
		return nil, err
	}

	if cfg.GitLab == nil {
		// Return a default but working config
		return &minderv1.GitLabProviderConfig{}, nil
	}

	return cfg.GitLab, nil
}

// MarshalV1Config marshals and validates the given config
// so it can safely be stored in the database
func MarshalV1Config(rawCfg json.RawMessage) (json.RawMessage, error) {
	var w glConfigWrapper
	if err := json.Unmarshal(rawCfg, &w); err != nil {
		return nil, err
	}

	// TODO: Add validation
	// err := w.GitLab.Validate()
	// if err != nil {
	// 	return nil, fmt.Errorf("error validating gitlab config: %w", err)
	// }

	return json.Marshal(w)
}

// CanImplement returns true if the provider can implement the given trait
func (_ *gitlabClient) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_GIT ||
		trait == minderv1.ProviderType_PROVIDER_TYPE_REST
}

func (c *gitlabClient) GetCredential() provifv1.GitLabCredential {
	return c.cred
}

// SupportsEntity implements the Provider interface
func (_ *gitlabClient) SupportsEntity(entType minderv1.Entity) bool {
	return entType == minderv1.Entity_ENTITY_REPOSITORIES
}

// RegisterEntity implements the Provider interface
func (c *gitlabClient) RegisterEntity(
	_ context.Context, entType minderv1.Entity, props *properties.Properties,
) (*properties.Properties, error) {
	if !c.SupportsEntity(entType) {
		return nil, errors.New("unsupported entity type")
	}

	// TODO: implement

	return props, nil
}

// DeregisterEntity implements the Provider interface
func (_ *gitlabClient) DeregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	// TODO: implement
	return nil
}
