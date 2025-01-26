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

	"github.com/mindersec/minder/internal/entities/properties"
	pbinternal "github.com/mindersec/minder/internal/proto"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// FormatPullRequestUpstreamID returns the upstream ID for a gitlab merge request
// This is done so we don't have to deal with conversions in the provider
// when dealing with entities
func FormatPullRequestUpstreamID(id int) string {
	return fmt.Sprintf("%d", id)
}

//nolint:gocyclo // TODO: Refactor to reduce complexity
func (c *gitlabClient) getPropertiesForPullRequest(
	ctx context.Context, getByProps *properties.Properties,
) (*properties.Properties, error) {
	uid, err := getByProps.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("upstream ID not found or invalid: %w", err)
	}

	iid, err := getByProps.GetProperty(PullRequestNumber).AsString()
	if err != nil {
		return nil, fmt.Errorf("merge request number not found or invalid: %w", err)
	}

	pid, err := getByProps.GetProperty(PullRequestProjectID).AsString()
	if err != nil {
		return nil, fmt.Errorf("project ID not found or invalid: %w", err)
	}

	mrURLPath, err := url.JoinPath("projects", pid, "merge_requests", iid)
	if err != nil {
		return nil, fmt.Errorf("failed to join URL path for merge request using upstream ID: %w", err)
	}

	// NOTE: We're not using github.com/xanzy/go-gitlab to do the actual
	// request here because of the way they form authentication for requests.
	// It would be ideal to use it, so we should consider contributing and making
	// that part more pluggable.
	req, err := c.NewRequest("GET", mrURLPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, provifv1.ErrEntityNotFound
		}

		return nil, fmt.Errorf("failed to get merge request: %s", resp.Status)
	}

	mr := &gitlab.MergeRequest{}
	if err := json.NewDecoder(resp.Body).Decode(mr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Validate - merge request upstream ID must match the one we requested
	if res := FormatPullRequestUpstreamID(mr.ID); res != uid {
		return nil, fmt.Errorf("merge request ID mismatch: %s != %s", res, uid)
	}

	proj, err := c.getGitLabProject(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	targetproj := proj
	if mr.SourceProjectID != 0 && mr.SourceProjectID != proj.ID {
		targetproj, err = c.getGitLabProject(ctx, FormatRepositoryUpstreamID(mr.SourceProjectID))
		if err != nil {
			return nil, fmt.Errorf("failed to get target project: %w", err)
		}
	}

	outProps, err := gitlabMergeRequestToProperties(mr, proj, targetproj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert merge request to properties: %w", err)
	}

	return outProps, nil
}

func gitlabMergeRequestToProperties(
	mr *gitlab.MergeRequest, proj *gitlab.Project, targetproj *gitlab.Project) (*properties.Properties, error) {
	ns, err := getGitlabProjectNamespace(proj)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	projName := proj.Name

	outProps, err := properties.NewProperties(map[string]any{
		// Unique upstream ID for the merge request
		properties.PropertyUpstreamID:           FormatPullRequestUpstreamID(mr.ID),
		properties.PropertyName:                 formatPullRequestName(ns, projName, FormatPullRequestUpstreamID(mr.IID)),
		properties.PullRequestCommitSHA:         mr.SHA,
		properties.PullRequestBaseCloneURL:      proj.HTTPURLToRepo,
		properties.PullRequestBaseDefaultBranch: mr.TargetBranch,
		properties.PullRequestTargetCloneURL:    targetproj.HTTPURLToRepo,
		properties.PullRequestTargetBranch:      mr.SourceBranch,
		properties.PullRequestUpstreamURL:       mr.WebURL,
		RepoPropertyNamespace:                   ns,
		RepoPropertyProjectName:                 projName,
		// internal ID of the merge request
		PullRequestNumber:    FormatPullRequestUpstreamID(mr.IID),
		PullRequestProjectID: FormatRepositoryUpstreamID(proj.ID),
		PullRequestAuthor:    int64(mr.Author.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create properties: %w", err)
	}

	return outProps, nil
}

func pullRequestV1FromProperties(prProps *properties.Properties) (*pbinternal.PullRequest, error) {
	_, err := prProps.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("failed to get upstream ID: %w", err)
	}

	iid, err := getStringProp(prProps, PullRequestNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request number: %w", err)
	}

	ns, err := getStringProp(prProps, RepoPropertyNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	projName, err := getStringProp(prProps, RepoPropertyProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get project name: %w", err)
	}

	commitSha, err := getStringProp(prProps, properties.PullRequestCommitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit SHA: %w", err)
	}

	mrURL, err := getStringProp(prProps, properties.PullRequestUpstreamURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request URL: %w", err)
	}

	authorID, err := prProps.GetProperty(PullRequestAuthor).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("failed to get author ID: %w", err)
	}

	basecloneurl := prProps.GetProperty(properties.PullRequestBaseCloneURL).GetString()
	targetcloneurl := prProps.GetProperty(properties.PullRequestTargetCloneURL).GetString()
	basebranch := prProps.GetProperty(properties.PullRequestBaseDefaultBranch).GetString()
	targetbranch := prProps.GetProperty(properties.PullRequestTargetBranch).GetString()

	// parse UpstreamID to int64
	id, err := strconv.ParseInt(iid, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse upstream ID: %w", err)
	}

	pbPR := &pbinternal.PullRequest{
		Number:         id,
		RepoOwner:      ns,
		RepoName:       projName,
		CommitSha:      commitSha,
		AuthorId:       authorID,
		Url:            mrURL,
		BaseCloneUrl:   basecloneurl,
		TargetCloneUrl: targetcloneurl,
		BaseRef:        basebranch,
		TargetRef:      targetbranch,
		Properties:     prProps.ToProtoStruct(),
	}

	return pbPR, nil
}

func getPullRequestNameFromProperties(props *properties.Properties) (string, error) {
	groupName, err := getStringProp(props, RepoPropertyNamespace)
	if err != nil {
		return "", err
	}

	projectName, err := getStringProp(props, RepoPropertyProjectName)
	if err != nil {
		return "", err
	}

	iid, err := getStringProp(props, PullRequestNumber)
	if err != nil {
		return "", err
	}

	return formatPullRequestName(groupName, projectName, iid), nil
}

func formatPullRequestName(groupName, projectName, iid string) string {
	return fmt.Sprintf("%s/%s/%s", groupName, projectName, iid)
}
