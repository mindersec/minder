// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	providerDocsBaseURL = "https://docs.mindersec.dev"
	providerDocsURL     = providerDocsBaseURL + "/understand/providers"
)

func (d *dockerHubImageLister) ProviderClassInfo() *minderv1.ProviderClassInfo {
	return &minderv1.ProviderClassInfo{
		Class:                  DockerHub,
		DisplayName:            "Docker Hub",
		Description:            "Docker Hub registry provider for image and OCI interactions.",
		SupportedProviderTypes: provifv1.ProviderTypesFromImpl(d),
		SupportedAuthFlows: []minderv1.AuthorizationFlow{
			minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
		},
		SupportedEntities: nil,
		DocumentationUrl:  providerDocsURL,
	}
}

// ClassInfo returns metadata for the Docker Hub provider class.
// It uses a nil-pointer receiver to avoid needing a live client instance.
func ClassInfo() *minderv1.ProviderClassInfo {
	return (*dockerHubImageLister)(nil).ProviderClassInfo()
}
