// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quay

import (
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	providerDocsBaseURL = "https://docs.mindersec.dev"
	providerDocsURL     = providerDocsBaseURL + "/understand/providers"
)

func (q *quayImageLister) ProviderClassInfo() *minderv1.ProviderClassInfo {
	return &minderv1.ProviderClassInfo{
		Class:                  Quay,
		DisplayName:            "Quay.io",
		Description:            "Quay.io registry provider for image and OCI interactions.",
		SupportedProviderTypes: provifv1.ProviderTypesFromImpl(q),
		SupportedAuthFlows: []minderv1.AuthorizationFlow{
			minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
		},
		SupportedEntities: nil,
		DocumentationUrl:  providerDocsURL,
	}
}

// ClassInfo returns metadata for the Quay.io provider class.
// It uses a nil-pointer receiver to avoid needing a live client instance.
func ClassInfo() *minderv1.ProviderClassInfo {
	return (*quayImageLister)(nil).ProviderClassInfo()
}
