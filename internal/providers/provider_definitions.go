// Copyright 2024 Stacklok, Inc.
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

package providers

import (
	"fmt"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/dockerhub"
	ghclient "github.com/stacklok/minder/internal/providers/github/clients"
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
}

// GetProviderClassDefinition returns the provider definition for the given provider class
func GetProviderClassDefinition(class string) (ProviderClassDefinition, error) {
	def, ok := supportedProviderClassDefinitions[class]
	if !ok {
		return ProviderClassDefinition{}, fmt.Errorf("provider %s not found", class)
	}
	return def, nil
}
