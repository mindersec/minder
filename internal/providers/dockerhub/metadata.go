// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	providerDocsBaseURL = "https://docs.mindersec.dev"
	providerDocsURL     = providerDocsBaseURL + "/understand/providers"
)

// ClassInfo returns metadata for the Docker Hub provider class.
func ClassInfo() *minderv1.ProviderClassInfo {
	return &minderv1.ProviderClassInfo{
		Class:       DockerHub,
		DisplayName: "Docker Hub",
		Description: "Docker Hub registry provider for image and OCI interactions.",
		SupportedProviderTypes: []minderv1.ProviderType{
			minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER,
			minderv1.ProviderType_PROVIDER_TYPE_OCI,
		},
		SupportedAuthFlows: []minderv1.AuthorizationFlow{
			minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
		},
		SupportedEntities: nil,
		DocumentationUrl:  providerDocsURL,
	}
}

func (*dockerHubImageLister) ProviderClassInfo() *minderv1.ProviderClassInfo {
	return ClassInfo()
}
