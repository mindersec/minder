// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestClassInfo(t *testing.T) {
	t.Parallel()

	info := ClassInfo()
	require.NotNil(t, info)

	assert.Equal(t, Class, info.Class)
	assert.Equal(t, "GitLab", info.DisplayName)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, providerDocsURL, info.DocumentationUrl)

	assert.ElementsMatch(t, []minderv1.AuthorizationFlow{
		minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
		minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW,
	}, info.SupportedAuthFlows)

	assert.ElementsMatch(t, []minderv1.Entity{
		minderv1.Entity_ENTITY_REPOSITORIES,
		minderv1.Entity_ENTITY_PULL_REQUESTS,
		minderv1.Entity_ENTITY_RELEASE,
	}, info.SupportedEntities)

	assert.ElementsMatch(t, []minderv1.ProviderType{
		minderv1.ProviderType_PROVIDER_TYPE_GIT,
		minderv1.ProviderType_PROVIDER_TYPE_REST,
		minderv1.ProviderType_PROVIDER_TYPE_REPO_LISTER,
	}, info.SupportedProviderTypes)
}
