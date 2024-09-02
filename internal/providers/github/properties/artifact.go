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

// Package properties provides utility functions for fetching and managing properties
package properties

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	go_github "github.com/google/go-github/v63/github"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/verifier/verifyif"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// ArtifactPropertyOwner is the owner of the artifact
	ArtifactPropertyOwner = "github/owner"
	// ArtifactPropertyName is the name of the artifact
	ArtifactPropertyName = "github/name"
	// ArtifactPropertyCreatedAt is the time the artifact was created
	ArtifactPropertyCreatedAt = "github/created_at"
	// ArtifactPropertyRepoOwner is the owner of the repository the artifact is in
	ArtifactPropertyRepoOwner = "github/repo_owner"
	// ArtifactPropertyRepoName is the name of the repository the artifact is in
	ArtifactPropertyRepoName = "github/repo_name"
	// ArtifactPropertyRepo is the full name of the repository the artifact is in
	ArtifactPropertyRepo = "github/repo"
	// ArtifactPropertyType is the type of the artifact
	ArtifactPropertyType = "github/type"
	// ArtifactPropertyVisibility is the visibility of the artifact
	ArtifactPropertyVisibility = "github/visibility"
)

// ArtifactFetcher fetches artifact properties
type ArtifactFetcher struct {
	propertyFetcherBase
}

// NewArtifactFetcher creates a new ArtifactFetcher
func NewArtifactFetcher() *ArtifactFetcher {
	return &ArtifactFetcher{
		propertyFetcherBase: propertyFetcherBase{
			propertyOrigins: []propertyOrigin{
				{
					keys: []string{
						// general entity
						properties.PropertyName,
						properties.PropertyUpstreamID,
						// github-specific
						ArtifactPropertyName,
						ArtifactPropertyOwner,
						ArtifactPropertyCreatedAt,
						ArtifactPropertyRepoOwner,
						ArtifactPropertyRepoName,
						ArtifactPropertyRepo,
						ArtifactPropertyType,
						ArtifactPropertyVisibility,
					},
					wrapper: getArtifactWrapper,
				},
			},
			operationalProperties: []string{},
		},
	}
}

// GetName returns the name of the artifact
func (_ *ArtifactFetcher) GetName(props *properties.Properties) (string, error) {
	// it seems like the previous code handles the case where owner is not set,
	// although it's not clear why it's necessary. Let's keep it for now, sigh.
	owner := props.GetProperty(ArtifactPropertyOwner).GetString()

	name, err := props.GetProperty(ArtifactPropertyName).AsString()
	if err != nil {
		return "", fmt.Errorf("failed to get artifact name: %w", err)
	}

	return getNameFromParams(owner, name), nil
}

func getNameFromParams(owner, name string) string {
	var prefix string
	if owner != "" {
		prefix = owner + "/"
	}

	return prefix + name
}

func parseArtifactName(name string) (string, string, string, error) {
	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], string(verifyif.ArtifactTypeContainer), nil
	} else if len(parts) == 1 {
		return "", parts[0], string(verifyif.ArtifactTypeContainer), nil
	}

	return "", "", "", fmt.Errorf("invalid name format")
}

func getArtifactWrapper(
	ctx context.Context, ghCli *go_github.Client, isOrg bool, getByProps *properties.Properties,
) (map[string]any, error) {
	owner, name, pkgType, err := getArtifactWrapperAttrsFromProps(getByProps)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact properties: %w", err)
	}
	fmt.Println(owner, name, pkgType)

	var fetchErr error
	var pkg *go_github.Package
	var result *go_github.Response
	if isOrg {
		pkg, result, fetchErr = ghCli.Organizations.GetPackage(ctx, owner, pkgType, name)
	} else {
		pkg, result, fetchErr = ghCli.Users.GetPackage(ctx, owner, pkgType, name)
	}

	if fetchErr != nil {
		if result != nil && result.StatusCode == http.StatusNotFound {
			return nil, v1.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to fetch package: %w", fetchErr)
	}

	return map[string]any{
		// general entity
		properties.PropertyUpstreamID: strconv.FormatInt(pkg.GetID(), 10),
		properties.PropertyName:       getNameFromParams(owner, name),
		// github-specific
		ArtifactPropertyName:       pkg.GetName(),
		ArtifactPropertyOwner:      pkg.GetOwner().GetLogin(),
		ArtifactPropertyCreatedAt:  pkg.GetCreatedAt().Format(time.RFC3339),
		ArtifactPropertyRepoOwner:  pkg.GetRepository().GetOwner().GetLogin(),
		ArtifactPropertyRepoName:   pkg.GetRepository().GetName(),
		ArtifactPropertyRepo:       pkg.GetRepository().GetFullName(),
		ArtifactPropertyType:       strings.ToLower(pkg.GetPackageType()),
		ArtifactPropertyVisibility: pkg.GetVisibility(),
	}, nil
}

func getArtifactWrapperAttrsFromProps(props *properties.Properties) (string, string, string, error) {
	ownerP := props.GetProperty(ArtifactPropertyOwner)
	nameP := props.GetProperty(ArtifactPropertyName)
	pkgTypeP := props.GetProperty(ArtifactPropertyType)
	if ownerP != nil && nameP != nil && pkgTypeP != nil {
		return ownerP.GetString(), nameP.GetString(), pkgTypeP.GetString(), nil
	}

	pkgNameP := props.GetProperty(properties.PropertyName)
	if pkgNameP != nil {
		return parseArtifactName(pkgNameP.GetString())
	}

	return "", "", "", fmt.Errorf("missing required properties")
}

// ArtifactV1FromProperties creates a minder v1 artifact from properties
func ArtifactV1FromProperties(props *properties.Properties) (*minderv1.Artifact, error) {
	upstreamId, err := props.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact upstream ID: %w", err)
	}

	parsedTime, err := time.Parse(time.RFC3339, props.GetProperty(ArtifactPropertyCreatedAt).GetString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at time: %w", err)
	}

	return &minderv1.Artifact{
		ArtifactPk: upstreamId,
		// the previous code also uses short names for artifact and the ingester relies on it
		Name:      props.GetProperty(ArtifactPropertyName).GetString(),
		Type:      props.GetProperty(ArtifactPropertyType).GetString(),
		CreatedAt: timestamppb.New(parsedTime),
		// the previous code also uses short names for repo and the ingester relies on it
		Repository: props.GetProperty(ArtifactPropertyRepoName).GetString(),
		Owner:      props.GetProperty(ArtifactPropertyOwner).GetString(),
		Visibility: props.GetProperty(ArtifactPropertyVisibility).GetString(),
	}, nil
}
