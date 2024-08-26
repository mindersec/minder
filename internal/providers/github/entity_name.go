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

package github

import (
	"errors"
	"fmt"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// GetEntityName implements the Provider interface
func (_ *GitHub) GetEntityName(entType minderv1.Entity, props *properties.Properties) (string, error) {
	return getEntityName(entType, props)
}

func getEntityName(entType minderv1.Entity, props *properties.Properties) (string, error) {
	if props == nil {
		return "", errors.New("properties are nil")
	}

	//nolint:exhaustive // we want to fail if we don't support the entity type
	switch entType {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return getRepoName(props)
	case minderv1.Entity_ENTITY_ARTIFACTS:
		return getArtifactName(props)
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return getPullRequestName(props)
	default:
		return "", fmt.Errorf("unsupported entity type: %s", entType)
	}
}

func getRepoName(props *properties.Properties) (string, error) {
	repoNameP := props.GetProperty(RepoPropertyName)
	repoOwnerP := props.GetProperty(RepoPropertyOwner)

	if repoNameP == nil || repoOwnerP == nil {
		return "", errors.New("missing required properties")
	}

	repoName := repoNameP.GetString()
	if repoName == "" {
		return "", errors.New("missing required repo-name property value")
	}

	repoOwner := repoOwnerP.GetString()
	if repoOwner == "" {
		return "", errors.New("missing required repo-owner property value")
	}

	return fmt.Sprintf("%s/%s", repoOwner, repoName), nil
}

func getArtifactName(_ *properties.Properties) (string, error) {
	// TODO: implement
	return "", errors.New("not implemented")
}

func getPullRequestName(_ *properties.Properties) (string, error) {
	// TODO: implement
	return "", errors.New("not implemented")
}
