// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package properties

import (
	"context"
	"fmt"
	"time"

	go_github "github.com/google/go-github/v63/github"

	"github.com/mindersec/minder/pkg/entities/properties"
)

// OrganizationFetcher is a GhPropertyFetcher for organizations
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
						properties.PropertyUpstreamID,
						properties.PropertyName,
						properties.OrgPropertyIsUser,
						properties.OrgPropertyHasOrganizationProjects,
						properties.OrgPropertyCreatedAt,
						properties.OrgPropertyPlanName,
					},
					wrapper: fetchOrganizationProperties,
				},
			},
		},
	}
}

// GetName returns the name of the organization
func (*OrganizationFetcher) GetName(props *properties.Properties) (string, error) {
	name := props.GetProperty(properties.PropertyName).GetString()
	if name == "" {
		return "", fmt.Errorf("missing property: %s", properties.PropertyName)
	}
	return name, nil
}

func fetchOrganizationProperties(
	ctx context.Context, ghCli *go_github.Client, _ bool, lookupProperties *properties.Properties,
) (map[string]any, error) {
	// We can look up by either exact upstream ID or by name (login).
	var user *go_github.User
	var err error

	upstreamIDProp := lookupProperties.GetProperty(properties.PropertyUpstreamID)
	nameProp := lookupProperties.GetProperty(properties.PropertyName)

	if upstreamIDProp != nil {
		id, parseErr := upstreamIDProp.AsInt64()
		if parseErr != nil {
			return nil, fmt.Errorf("invalid upstream ID: %w", parseErr)
		}
		user, _, err = ghCli.Users.GetByID(ctx, id)
	} else if name := nameProp.GetString(); name != "" {
		user, _, err = ghCli.Users.Get(ctx, name)
	} else {
		return nil, fmt.Errorf("either upstream_id or name (login) must be provided to fetch an organization")
	}

	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, fmt.Errorf("organization/user not found")
	}

	result := map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(user.GetID()),
		properties.PropertyName:       user.GetLogin(),
		properties.OrgPropertyIsUser:  user.GetType() == "User",
	}

	if user.GetType() == "Organization" {
		org, _, err := ghCli.Organizations.GetByID(ctx, user.GetID())
		if err == nil {
			if org.HasOrganizationProjects != nil {
				result[properties.OrgPropertyHasOrganizationProjects] = org.GetHasOrganizationProjects()
			}
		}
	}
	if user.CreatedAt != nil {
		result[properties.OrgPropertyCreatedAt] = user.GetCreatedAt().Format(time.RFC3339)
	}
	if user.Plan != nil {
		result[properties.OrgPropertyPlanName] = user.GetPlan().GetName()
	}

	return result, nil
}
