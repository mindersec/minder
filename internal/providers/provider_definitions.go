// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"fmt"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/dockerhub"
	ghclient "github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/providers/gitlab"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	providerDocsBaseURL   = "https://mindersec.github.io/docs"
	providerDocsGeneric   = providerDocsBaseURL + "/understand/providers"
	providerDocsGitHubURL = providerDocsBaseURL + "/integrations/provider_integrations/github"
)

// ProviderClassDefinition contains the static fields needed when creating a provider
type ProviderClassDefinition struct {
	DisplayName        string
	Description        string
	DocumentationURL   string
	CreationHelp       string
	Traits             []db.ProviderType
	AuthorizationFlows []db.AuthorizationFlow
	SupportedEntities  []minderv1.Entity
}

var supportedProviderClassDefinitions = map[string]ProviderClassDefinition{
	ghclient.GithubApp: {
		DisplayName:        "GitHub App",
		Description:        "GitHub App-based provider with installation-based authorization.",
		DocumentationURL:   providerDocsGitHubURL,
		CreationHelp:       "Install the configured GitHub App and enroll using --class github-app.",
		Traits:             ghclient.AppImplements,
		AuthorizationFlows: ghclient.AppAuthorizationFlows,
		SupportedEntities: []minderv1.Entity{
			minderv1.Entity_ENTITY_REPOSITORIES,
			minderv1.Entity_ENTITY_PULL_REQUESTS,
			minderv1.Entity_ENTITY_ARTIFACTS,
			minderv1.Entity_ENTITY_RELEASE,
		},
	},
	ghclient.Github: {
		DisplayName:        "GitHub OAuth",
		Description:        "GitHub provider using OAuth credentials.",
		DocumentationURL:   providerDocsGitHubURL,
		CreationHelp:       "Authorize GitHub access through the OAuth flow and enroll using --class github.",
		Traits:             ghclient.OAuthImplements,
		AuthorizationFlows: ghclient.OAuthAuthorizationFlows,
		SupportedEntities: []minderv1.Entity{
			minderv1.Entity_ENTITY_REPOSITORIES,
			minderv1.Entity_ENTITY_PULL_REQUESTS,
			minderv1.Entity_ENTITY_ARTIFACTS,
			minderv1.Entity_ENTITY_RELEASE,
		},
	},
	dockerhub.DockerHub: {
		DisplayName:        "Docker Hub",
		Description:        "Docker Hub registry provider for image and OCI interactions.",
		DocumentationURL:   providerDocsGeneric,
		CreationHelp:       "Provide Docker Hub credentials and enroll using --class dockerhub.",
		Traits:             dockerhub.Implements,
		AuthorizationFlows: dockerhub.AuthorizationFlows,
		SupportedEntities:  nil,
	},
	gitlab.Class: {
		DisplayName:        "GitLab",
		Description:        "GitLab provider using OAuth credentials.",
		DocumentationURL:   providerDocsGeneric,
		CreationHelp:       "Authorize GitLab access through the OAuth flow and enroll using --class gitlab.",
		Traits:             gitlab.Implements,
		AuthorizationFlows: gitlab.AuthorizationFlows,
		SupportedEntities: []minderv1.Entity{
			minderv1.Entity_ENTITY_REPOSITORIES,
			minderv1.Entity_ENTITY_PULL_REQUESTS,
			minderv1.Entity_ENTITY_RELEASE,
		},
	},
}

// ListProviderClassDefinitions returns all known provider class definitions.
func ListProviderClassDefinitions() map[string]ProviderClassDefinition {
	out := make(map[string]ProviderClassDefinition, len(supportedProviderClassDefinitions))
	for class, definition := range supportedProviderClassDefinitions {
		out[class] = definition
	}

	return out
}

// GetProviderClassDefinition returns the provider definition for the given provider class
func GetProviderClassDefinition(class string) (ProviderClassDefinition, error) {
	def, ok := supportedProviderClassDefinitions[class]
	if !ok {
		return ProviderClassDefinition{}, fmt.Errorf("provider %s not found", class)
	}
	return def, nil
}
