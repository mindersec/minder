// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	testhelper "github.com/mindersec/minder/pkg/providers/v1/testing"
)

func TestRegistration(t *testing.T) {
	t.Parallel()
	// We don't need a full constructor here, so we're naughty
	q := &quayImageLister{}
	testhelper.CheckRegistrationExcept(t, q)
}

func TestClassInfo(t *testing.T) {
	t.Parallel()

	info := ClassInfo()
	require.NotNil(t, info)

	assert.Equal(t, Quay, info.Class)
	assert.Equal(t, "Quay.io", info.DisplayName)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, providerDocsURL, info.DocumentationUrl)

	assert.ElementsMatch(t, []minderv1.AuthorizationFlow{
		minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
	}, info.SupportedAuthFlows)

	assert.Empty(t, info.SupportedEntities)

	assert.ElementsMatch(t, []minderv1.ProviderType{
		minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER,
		minderv1.ProviderType_PROVIDER_TYPE_OCI,
	}, info.SupportedProviderTypes)
}
