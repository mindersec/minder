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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/xanzy/go-gitlab"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func (c *gitlabClient) getPropertiesForRepo(
	ctx context.Context, getByProps *properties.Properties,
) (*properties.Properties, error) {
	uid, err := getByProps.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("upstream ID not found or invalid: %w", err)
	}

	projectURLPath, err := url.JoinPath("projects", url.PathEscape(uid))
	if err != nil {
		return nil, fmt.Errorf("failed to join URL path for project using upstream ID: %w", err)
	}

	// NOTE: We're not using github.com/xanzy/go-gitlab to do the actual
	// request here because of the way they form authentication for requests.
	// It would be ideal to use it, so we should consider contributing and making
	// that part more pluggable.
	req, err := c.NewRequest("GET", projectURLPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, provifv1.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to get projects: %s", resp.Status)
	}

	proj := &gitlab.Project{}
	if err := json.NewDecoder(resp.Body).Decode(proj); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	outProps, err := gitlabProjectToProperties(proj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert project to properties: %w", err)
	}

	return getByProps.Merge(outProps), nil
}

func gitlabProjectToProperties(proj *gitlab.Project) (*properties.Properties, error) {
	ns := proj.Namespace
	if ns == nil {
		return nil, fmt.Errorf("gitlab project %d has no namespace", proj.ID)
	}
	owner := ns.Path

	var license string
	if proj.License != nil {
		license = proj.License.Name
	}

	outProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID:     FormatRepositoryUpstreamID(proj.ID),
		properties.PropertyName:           formatRepoName(owner, proj.Name),
		properties.RepoPropertyIsPrivate:  proj.Visibility == gitlab.PrivateVisibility,
		properties.RepoPropertyIsArchived: proj.Archived,
		properties.RepoPropertyIsFork:     proj.ForkedFromProject != nil,
		RepoPropertyDefaultBranch:         proj.DefaultBranch,
		RepoPropertyNamespace:             owner,
		RepoPropertyProjectName:           proj.Name,
		RepoPropertyLicense:               license,
		RepoPropertyCloneURL:              proj.HTTPURLToRepo,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create properties: %w", err)
	}

	return outProps, nil
}

func repoV1FromProperties(repoProperties *properties.Properties) (*minderv1.Repository, error) {
	upstreamID, err := repoProperties.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("error fetching upstream ID property: %w", err)
	}

	// convert the upstream ID to an int64
	repoId, err := strconv.ParseInt(upstreamID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error converting upstream ID to int64: %w", err)
	}

	name, err := repoProperties.GetProperty(RepoPropertyProjectName).AsString()
	if err != nil {
		return nil, fmt.Errorf("error fetching project property: %w", err)
	}

	owner, err := repoProperties.GetProperty(RepoPropertyNamespace).AsString()
	if err != nil {
		return nil, fmt.Errorf("error fetching namespace property: %w", err)
	}

	isPrivate, err := repoProperties.GetProperty(properties.RepoPropertyIsPrivate).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_private property: %w", err)
	}

	isFork, err := repoProperties.GetProperty(properties.RepoPropertyIsFork).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_fork property: %w", err)
	}

	pbRepo := &minderv1.Repository{
		Name:          name,
		Owner:         owner,
		RepoId:        repoId,
		CloneUrl:      repoProperties.GetProperty(RepoPropertyCloneURL).GetString(),
		IsPrivate:     isPrivate,
		IsFork:        isFork,
		DefaultBranch: repoProperties.GetProperty(RepoPropertyDefaultBranch).GetString(),
		License:       repoProperties.GetProperty(RepoPropertyLicense).GetString(),
		Properties:    repoProperties.ToProtoStruct(),
	}

	return pbRepo, nil
}
