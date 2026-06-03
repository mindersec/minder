// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	providerDocsBaseURL = "https://docs.mindersec.dev"
	providerDocsURL     = providerDocsBaseURL + "/understand/providers"
)

func (c *gitlabClient) ProviderClassInfo() *minderv1.ProviderClassInfo {
	return &minderv1.ProviderClassInfo{
		Class:                  Class,
		DisplayName:            "GitLab",
		Description:            "GitLab provider using OAuth credentials.",
		SupportedProviderTypes: provifv1.ProviderTypesFromImpl(c),
		SupportedAuthFlows: []minderv1.AuthorizationFlow{
			minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
			minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW,
		},
		SupportedEntities: []minderv1.Entity{
			minderv1.Entity_ENTITY_REPOSITORIES,
			minderv1.Entity_ENTITY_PULL_REQUESTS,
			minderv1.Entity_ENTITY_RELEASE,
		},
		DocumentationUrl: providerDocsURL,
	}
}

// ClassInfo returns metadata for the GitLab provider class.
// It uses a nil-pointer receiver to avoid needing a live client instance.
func ClassInfo() *minderv1.ProviderClassInfo {
	return (*gitlabClient)(nil).ProviderClassInfo()
}
