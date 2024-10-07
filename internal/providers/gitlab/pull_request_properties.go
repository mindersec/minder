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

	outProps, err := gitlabMergeRequestToProperties(mr, proj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert merge request to properties: %w", err)
	}

	return outProps, nil
}

func gitlabMergeRequestToProperties(mr *gitlab.MergeRequest, proj *gitlab.Project) (*properties.Properties, error) {
	ns, err := getGitlabProjectNamespace(proj)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	projName := proj.Name

	outProps, err := properties.NewProperties(map[string]any{
		// Unique upstream ID for the merge request
		properties.PropertyUpstreamID: FormatPullRequestUpstreamID(mr.ID),
		properties.PropertyName:       formatPullRequestName(ns, projName, FormatPullRequestUpstreamID(mr.IID)),
		RepoPropertyNamespace:         ns,
		RepoPropertyProjectName:       projName,
		// internal ID of the merge request
		PullRequestNumber:       FormatPullRequestUpstreamID(mr.IID),
		PullRequestProjectID:    FormatRepositoryUpstreamID(proj.ID),
		PullRequestSourceBranch: mr.SourceBranch,
		PullRequestTargetBranch: mr.TargetBranch,
		PullRequestCommitSHA:    mr.SHA,
		PullRequestAuthor:       int64(mr.Author.ID),
		PullRequestURL:          mr.WebURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create properties: %w", err)
	}

	return outProps, nil
}

func pullRequestV1FromProperties(prProps *properties.Properties) (*minderv1.PullRequest, error) {
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

	commitSha, err := getStringProp(prProps, PullRequestCommitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit SHA: %w", err)
	}

	mrURL, err := getStringProp(prProps, PullRequestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request URL: %w", err)
	}

	authorID, err := prProps.GetProperty(PullRequestAuthor).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("failed to get author ID: %w", err)
	}

	// parse UpstreamID to int64
	id, err := strconv.ParseInt(iid, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse upstream ID: %w", err)
	}

	pbPR := &minderv1.PullRequest{
		Number:     id,
		RepoOwner:  ns,
		RepoName:   projName,
		CommitSha:  commitSha,
		AuthorId:   authorID,
		Url:        mrURL,
		Properties: prProps.ToProtoStruct(),
	}

	return pbPR, nil
}
