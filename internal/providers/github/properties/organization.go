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

// Organization Properties
const (
	// OrganizationPropertyOwner represents the github owner
	OrganizationPropertyOwner = "github/owner"
	// OrganizationPropertyWebsite represents the github website
	OrganizationPropertyWebsite = "github/website"
)

// OrganizationFetcher is a property fetcher for organizations
type OrganizationFetcher struct {
	propertyFetcherBase
}

// NewOrganizationFetcher creates a new OrganizationFetcher
func NewOrganizationFetcher() *OrganizationFetcher {
	return &OrganizationFetcher{
		propertyFetcherBase: propertyFetcherBase{
			propertyOrigins: []propertyOrigin{
				{
					keys: []string{
						// general entity
						properties.PropertyName,
						properties.PropertyUpstreamID,
						// general organization
						OrganizationPropertyOwner,
					},
					wrapper: getOrganizationWrapper,
				},
			},
			operationalProperties: []string{},
		},
	}
}

// GetName returns the name of the release
func (_ *OrganizationFetcher) GetName(props *properties.Properties) (string, error) {
	owner := props.GetProperty(OrganizationPropertyOwner).GetString()

	return owner, nil
}

func getOrganizationWrapper(
	ctx context.Context, ghCli *go_github.Client, _ bool, getByProps *properties.Properties,
) (map[string]any, error) {
	owner, err := getByProps.GetProperty(OrganizationPropertyOwner).AsString()
	if err != nil {
		return nil, fmt.Errorf("owner not found or invalid: %w", err)
	}

	org, result, err := ghCli.Organizations.Get(ctx, owner)
	if err != nil {
		if result != nil && result.StatusCode == http.StatusNotFound {
			return nil, v1.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to fetch organization: %w", err)
	}

	props := map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(org.GetID()),
		properties.PropertyName:       owner,
		OrganizationPropertyOwner:     owner,
	}

	if org.GetBlog() != "" {
		props[OrganizationPropertyWebsite] = org.GetBlog()
	}

	return props, nil
}

// EntityInstanceV1FromOrganizationProperties creates a new EntityInstance from the given properties
func EntityInstanceV1FromOrganizationProperties(props *properties.Properties) (*minderv1.EntityInstance, error) {
	owner := props.GetProperty(OrganizationPropertyOwner).GetString()

	name := owner

	return &minderv1.EntityInstance{
		Type:       minderv1.Entity_ENTITY_ORGANIZATION,
		Name:       name,
		Properties: props.ToProtoStruct(),
	}, nil
}
