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
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog"
	gitlablib "github.com/xanzy/go-gitlab"
	"golang.org/x/mod/semver"

	"github.com/mindersec/minder/internal/entities/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// FormatReleaseUpstreamID returns the upstream ID for a gitlab release
// This is done so we don't have to deal with conversions in the provider
// when dealing with entities
func FormatReleaseUpstreamID(id int) string {
	return fmt.Sprintf("%d", id)
}

func (c *gitlabClient) getPropertiesForRelease(
	ctx context.Context, getByProps *properties.Properties,
) (*properties.Properties, error) {
	uid, err := getByProps.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("upstream ID not found or invalid: %w", err)
	}

	pid, err := getByProps.GetProperty(ReleasePropertyProjectID).AsString()
	if err != nil {
		return nil, fmt.Errorf("project ID not found or invalid: %w", err)
	}

	releaseTagName, err := getByProps.GetProperty(ReleasePropertyTag).AsString()
	if err != nil {
		return nil, fmt.Errorf("tag name not found or invalid: %w", err)
	}

	releasePath, err := url.JoinPath("projects", pid, "releases", releaseTagName)
	if err != nil {
		return nil, fmt.Errorf("failed to join URL path for release using upstream ID: %w", err)
	}

	// NOTE: We're not using github.com/xanzy/go-gitlab to do the actual
	// request here because of the way they form authentication for requests.
	// It would be ideal to use it, so we should consider contributing and making
	// that part more pluggable.
	req, err := c.NewRequest(http.MethodGet, releasePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for release using upstream ID: %w", err)
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get release using upstream ID: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get release using upstream ID: %w", err)
	}

	release := &gitlablib.Release{}
	if err := json.NewDecoder(resp.Body).Decode(release); err != nil {
		return nil, fmt.Errorf("failed to decode response for release using upstream ID: %w", err)
	}

	proj, err := c.getGitLabProject(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Get the commit refs for the release
	refs, err := c.getCommitBranchRefs(ctx, pid, release.Commit.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit refs: %w", err)
	}

	if len(refs) == 0 {
		return nil, fmt.Errorf("no commit refs found for release")
	}

	// try to guess the branch from the commit refs
	branch := guessBranchFromCommitRefs(refs, release.TagName)

	outProps, err := gitlabReleaseToProperties(uid, releaseTagName, proj, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to convert release to properties: %w", err)
	}

	return outProps, nil
}

func (c *gitlabClient) getCommitBranchRefs(
	ctx context.Context, projID string, commitID string,
) ([]*gitlablib.CommitRef, error) {
	commitRefsPath, err := url.JoinPath("projects", projID, "repository", "commits", commitID, "refs")
	if err != nil {
		return nil, fmt.Errorf("failed to join URL path for commit refs: %w", err)
	}

	// append the type=branch query param
	// It's safe to append this way since it's a constant string
	// and we've already parsed the URL
	commitRefsPath = fmt.Sprintf("%s?type=branch", commitRefsPath)

	var resp *http.Response
	err = backoff.RetryNotify(
		func() error {
			req, err := c.NewRequest(http.MethodGet, commitRefsPath, nil)
			if err != nil {
				return fmt.Errorf("failed to create request for commit refs: %w", err)
			}

			resp, err = c.Do(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to get commit refs: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to get commit refs: HTTP Status %d", resp.StatusCode)
			}

			return nil
		},
		backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5),
		func(err error, _ time.Duration) {
			zerolog.Ctx(ctx).Debug().Err(err).Msg("failed to get commit refs, retrying")
		})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit refs: %w", err)
	}

	defer resp.Body.Close()

	commitRefs := []*gitlablib.CommitRef{}
	if err := json.NewDecoder(resp.Body).Decode(&commitRefs); err != nil {
		return nil, fmt.Errorf("failed to decode response for commit refs: %w", err)
	}

	return commitRefs, nil
}

func releaseEntityV1FromProperties(props *properties.Properties) (*minderv1.EntityInstance, error) {
	// validation
	_, err := props.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("upstream ID not found or invalid: %w", err)
	}

	_, err = props.GetProperty(ReleasePropertyProjectID).AsString()
	if err != nil {
		return nil, fmt.Errorf("project ID not found or invalid: %w", err)
	}

	_, err = props.GetProperty(ReleasePropertyBranch).AsString()
	if err != nil {
		return nil, fmt.Errorf("branch not found or invalid: %w", err)
	}

	if _, err = props.GetProperty(RepoPropertyNamespace).AsString(); err != nil {
		return nil, fmt.Errorf("namespace not found or invalid: %w", err)
	}

	if _, err = props.GetProperty(RepoPropertyProjectName).AsString(); err != nil {
		return nil, fmt.Errorf("project name not found or invalid: %w", err)
	}

	name, err := getReleaseNameFromProperties(props)
	if err != nil {
		return nil, fmt.Errorf("failed to get release name: %w", err)
	}

	return &minderv1.EntityInstance{
		Type:       minderv1.Entity_ENTITY_RELEASE,
		Name:       name,
		Properties: props.ToProtoStruct(),
	}, nil
}

func getReleaseNameFromProperties(props *properties.Properties) (string, error) {
	branch, err := props.GetProperty(ReleasePropertyTag).AsString()
	if err != nil {
		return "", fmt.Errorf("branch not found or invalid: %w", err)
	}

	ns, err := props.GetProperty(RepoPropertyNamespace).AsString()
	if err != nil {
		return "", fmt.Errorf("namespace not found or invalid: %w", err)
	}

	projName, err := props.GetProperty(RepoPropertyProjectName).AsString()
	if err != nil {
		return "", fmt.Errorf("project name not found or invalid: %w", err)
	}

	return formatReleaseName(ns, projName, branch), nil
}

func gitlabReleaseToProperties(
	releaseID string, releaseTag string, proj *gitlablib.Project, branch string,
) (*properties.Properties, error) {
	ns, err := getGitlabProjectNamespace(proj)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	projName := proj.Name

	return properties.NewProperties(map[string]interface{}{
		properties.PropertyUpstreamID: releaseID,
		ReleasePropertyProjectID:      FormatRepositoryUpstreamID(proj.ID),
		ReleasePropertyTag:            releaseTag,
		ReleasePropertyBranch:         branch,
		RepoPropertyNamespace:         ns,
		RepoPropertyProjectName:       projName,
	})
}

func guessBranchFromCommitRefs(refs []*gitlablib.CommitRef, tagName string) string {
	if len(refs) == 1 {
		return refs[0].Name
	}

	for _, ref := range refs {
		// simple, if the ref name is the same as the tag name, return it
		if ref.Name == tagName {
			return ref.Name
		}

		// if the ref name starts with release, return it
		if strings.HasPrefix(ref.Name, "release-") || strings.HasPrefix(ref.Name, "release/") {
			return ref.Name
		}

		// semver - match major and minor versions
		if semver.IsValid(ref.Name) && semver.IsValid(tagName) {
			if semver.Major(ref.Name) == semver.Major(tagName) {
				return ref.Name
			}
		}

		// if the ref name starts with the tag name, return it
		if strings.HasPrefix(ref.Name, tagName) {
			return ref.Name
		}

		if strings.HasPrefix(tagName, ref.Name) {
			return ref.Name
		}

		// if it's `main` or `master`, return it
		if ref.Name == "main" || ref.Name == "master" {
			return ref.Name
		}
	}

	// If we can't find a match, just return the first ref
	return refs[0].Name
}

func formatReleaseName(ns, projName, branch string) string {
	return fmt.Sprintf("%s/%s/%s", ns, projName, branch)
}
