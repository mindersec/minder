// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"fmt"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/dockerhub"
	ghclient "github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/providers/gitlab"
)

// ProviderClassDefinition contains the static fields needed when creating a provider
type ProviderClassDefinition struct {
	Traits             []db.ProviderType
	AuthorizationFlows []db.AuthorizationFlow
}

var supportedProviderClassDefinitions = map[string]ProviderClassDefinition{
	ghclient.GithubApp: {
		Traits:             ghclient.AppImplements,
		AuthorizationFlows: ghclient.AppAuthorizationFlows,
	},
	ghclient.Github: {
		Traits:             ghclient.OAuthImplements,
		AuthorizationFlows: ghclient.OAuthAuthorizationFlows,
	},
	dockerhub.DockerHub: {
		Traits:             dockerhub.Implements,
		AuthorizationFlows: dockerhub.AuthorizationFlows,
	},
	gitlab.Class: {
		Traits:             gitlab.Implements,
		AuthorizationFlows: gitlab.AuthorizationFlows,
	},
}

// GetProviderClassDefinition returns the provider definition for the given provider class
func GetProviderClassDefinition(class string) (ProviderClassDefinition, error) {
	def, ok := supportedProviderClassDefinitions[class]
	if !ok {
		return ProviderClassDefinition{}, fmt.Errorf("provider %s not found", class)
	}
	return def, nil
}
