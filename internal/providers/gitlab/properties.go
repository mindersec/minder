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

package gitlab

import (
	"context"
	"errors"
	"fmt"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	// RepoPropertyGroupName represents the gitlab group
	RepoPropertyGroupName = "gitlab/group_name"
	// RepoPropertyProjectName represents the gitlab project
	RepoPropertyProjectName = "gitlab/project_name"
)

// FetchAllProperties implements the provider interface
// TODO: Implement this
func (_ *gitlabClient) FetchAllProperties(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	return nil, nil
}

// FetchProperty implements the provider interface
// TODO: Implement this
func (_ *gitlabClient) FetchProperty(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

// GetEntityName implements the provider interface
func (_ *gitlabClient) GetEntityName(entityType minderv1.Entity, props *properties.Properties) (string, error) {
	if props == nil {
		return "", errors.New("properties are nil")
	}

	if entityType == minderv1.Entity_ENTITY_REPOSITORIES {
		groupName, err := getStringProp(props, RepoPropertyGroupName)
		if err != nil {
			return "", err
		}

		projectName, err := getStringProp(props, RepoPropertyProjectName)
		if err != nil {
			return "", err
		}

		return formatRepoName(groupName, projectName), nil
	}

	return "", fmt.Errorf("entity type %s not supported", entityType)
}

func getStringProp(props *properties.Properties, key string) (string, error) {
	value, err := props.GetProperty(key).AsString()
	if err != nil {
		return "", fmt.Errorf("property %s not found or not a string", key)
	}

	return value, nil
}

func formatRepoName(groupName, projectName string) string {
	return groupName + "/" + projectName
}
