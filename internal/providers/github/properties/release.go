// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package properties

import (
	"context"
	"fmt"
	"net/http"

	go_github "github.com/google/go-github/v63/github"

	"github.com/mindersec/minder/internal/entities/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
)

// Release Properties
const (
	// ReleasePropertyOwner represents the github owner
	ReleasePropertyOwner = "github/owner"
	// ReleasePropertyRepo represents the github repo
	ReleasePropertyRepo = "github/repo"
)

// ReleaseFetcher is a property fetcher for releases
type ReleaseFetcher struct {
	propertyFetcherBase
}

// NewReleaseFetcher creates a new ReleaseFetcher
func NewReleaseFetcher() *ReleaseFetcher {
	return &ReleaseFetcher{
		propertyFetcherBase: propertyFetcherBase{
			propertyOrigins: []propertyOrigin{
				{
					keys: []string{
						// general entity
						properties.PropertyName,
						properties.PropertyUpstreamID,
						// general release
						properties.ReleasePropertyTag,
						properties.ReleasePropertyBranch,
						ReleasePropertyOwner,
						ReleasePropertyRepo,
					},
					wrapper: getReleaseWrapper,
				},
			},
			operationalProperties: []string{},
		},
	}
}

// GetName returns the name of the release
func (_ *ReleaseFetcher) GetName(props *properties.Properties) (string, error) {
	owner := props.GetProperty(ReleasePropertyOwner).GetString()
	repo, err := props.GetProperty(ReleasePropertyRepo).AsString()
	if err != nil {
		return "", fmt.Errorf("failed to get repo name: %w", err)
	}

	tag, err := props.GetProperty(properties.ReleasePropertyTag).AsString()
	if err != nil {
		return "", fmt.Errorf("failed to get tag name: %w", err)
	}

	return getReleaseNameFromParams(owner, repo, tag), nil
}

func getReleaseNameFromParams(owner, repo, tag string) string {
	if owner == "" {
		return fmt.Sprintf("%s/%s", repo, tag)
	}

	return fmt.Sprintf("%s/%s/%s", owner, repo, tag)
}

func getReleaseWrapper(
	ctx context.Context, ghCli *go_github.Client, _ bool, getByProps *properties.Properties,
) (map[string]any, error) {
	// TODO: Should I be parsing this as string or int64?
	// if string, then I should convert it to int64
	upstreamID, err := getByProps.GetProperty(properties.PropertyUpstreamID).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("upstream ID not found or invalid: %w", err)
	}

	owner, err := getByProps.GetProperty(ReleasePropertyOwner).AsString()
	if err != nil {
		return nil, fmt.Errorf("owner not found or invalid: %w", err)
	}

	repo, err := getByProps.GetProperty(ReleasePropertyRepo).AsString()
	if err != nil {
		return nil, fmt.Errorf("repo not found or invalid: %w", err)
	}

	var fetchErr error
	var release *go_github.RepositoryRelease
	var result *go_github.Response
	release, result, fetchErr = ghCli.Repositories.GetRelease(ctx, owner, repo,
		upstreamID)
	if fetchErr != nil {
		if result != nil && result.StatusCode == http.StatusNotFound {
			return nil, v1.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to fetch release: %w", fetchErr)
	}

	return map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(release.GetID()),
		properties.PropertyName:       getReleaseNameFromParams(owner, repo, release.GetTagName()),
		ReleasePropertyOwner:          owner,
		ReleasePropertyRepo:           repo,
		properties.ReleasePropertyTag: release.GetTagName(),
	}, nil
}

// EntityInstanceV1FromReleaseProperties creates a new EntityInstance from the given properties
func EntityInstanceV1FromReleaseProperties(props *properties.Properties) (*minderv1.EntityInstance, error) {
	_, err := props.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("upstream ID not found or invalid: %w", err)
	}

	tag, err := props.GetProperty(properties.ReleasePropertyTag).AsString()
	if err != nil {
		return nil, fmt.Errorf("tag not found or invalid: %w", err)
	}

	_, err = props.GetProperty(properties.ReleasePropertyBranch).AsString()
	if err != nil {
		return nil, fmt.Errorf("branch not found or invalid: %w", err)
	}

	owner := props.GetProperty(ReleasePropertyOwner).GetString()

	repo, err := props.GetProperty(ReleasePropertyRepo).AsString()
	if err != nil {
		return nil, fmt.Errorf("repo not found or invalid: %w", err)
	}

	name := getReleaseNameFromParams(owner, repo, tag)

	return &minderv1.EntityInstance{
		Type:       minderv1.Entity_ENTITY_RELEASE,
		Name:       name,
		Properties: props.ToProtoStruct(),
	}, nil
}
