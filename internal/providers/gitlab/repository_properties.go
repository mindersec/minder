// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// FormatRepositoryUpstreamID returns the upstream ID for a gitlab project
// This is done so we don't have to deal with conversions in the provider
// when dealing with entities
func FormatRepositoryUpstreamID(id int) string {
	return fmt.Sprintf("%d", id)
}

func (c *gitlabClient) getPropertiesForRepo(
	ctx context.Context, getByProps *properties.Properties,
) (*properties.Properties, error) {
	uid, err := getByProps.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("upstream ID not found or invalid: %w", err)
	}

	proj, err := c.getGitLabProject(ctx, uid)
	if err != nil {
		return nil, err
	}

	outProps, err := gitlabProjectToProperties(proj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert project to properties: %w", err)
	}

	return getByProps.Merge(outProps), nil
}

func (c *gitlabClient) getGitLabProject(
	ctx context.Context, upstreamID string,
) (*gitlab.Project, error) {
	projectURLPath, err := url.JoinPath("projects", url.PathEscape(upstreamID))
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

	return proj, nil
}

func gitlabProjectToProperties(proj *gitlab.Project) (*properties.Properties, error) {
	owner, err := getGitlabProjectNamespace(proj)
	if err != nil {
		return nil, fmt.Errorf("failed to get project namespace: %w", err)
	}

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

func getGitlabProjectNamespace(proj *gitlab.Project) (string, error) {
	if proj.Namespace == nil {
		return "", fmt.Errorf("gitlab project %d has no namespace", proj.ID)
	}

	return proj.Namespace.Path, nil
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

func getRepoNameFromProperties(props *properties.Properties) (string, error) {
	groupName, err := getStringProp(props, RepoPropertyNamespace)
	if err != nil {
		return "", err
	}

	projectName, err := getStringProp(props, RepoPropertyProjectName)
	if err != nil {
		return "", err
	}

	return formatRepoName(groupName, projectName), nil
}

func formatRepoName(groupName, projectName string) string {
	return groupName + "/" + projectName
}
