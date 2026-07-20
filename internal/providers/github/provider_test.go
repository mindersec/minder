// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/github/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	testhelper "github.com/mindersec/minder/pkg/providers/v1/testing"
)

func TestRegistration(t *testing.T) {
	t.Parallel()
	// We don't need a full constructor here, so we're naughty
	gh := &GitHub{
		propertyFetchers: properties.NewPropertyFetcherFactory(),
	}
	// Repositories do a bunch of special registration, so skip them
	// in this test -- we test them in common_test.go.
	testhelper.CheckRegistrationExcept(t, gh, minderv1.Entity_ENTITY_REPOSITORIES)
}

func TestProviderClassInfo(t *testing.T) {
	t.Parallel()

	wantEntities := []minderv1.Entity{
		minderv1.Entity_ENTITY_REPOSITORIES,
		minderv1.Entity_ENTITY_PULL_REQUESTS,
		minderv1.Entity_ENTITY_ARTIFACTS,
		minderv1.Entity_ENTITY_RELEASE,
	}
	wantTypes := []minderv1.ProviderType{
		minderv1.ProviderType_PROVIDER_TYPE_GITHUB,
		minderv1.ProviderType_PROVIDER_TYPE_GIT,
		minderv1.ProviderType_PROVIDER_TYPE_REST,
		minderv1.ProviderType_PROVIDER_TYPE_REPO_LISTER,
		minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER,
	}

	tests := []struct {
		name          string
		providerClass db.ProviderClass
		wantClass     string
		wantDisplay   string
		wantAuthFlows []minderv1.AuthorizationFlow
	}{
		{
			name:          "GitHub OAuth",
			providerClass: db.ProviderClassGithub,
			wantClass:     string(db.ProviderClassGithub),
			wantDisplay:   "GitHub OAuth",
			wantAuthFlows: []minderv1.AuthorizationFlow{
				minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
				minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW,
			},
		},
		{
			name:          "GitHub App",
			providerClass: db.ProviderClassGithubApp,
			wantClass:     string(db.ProviderClassGithubApp),
			wantDisplay:   "GitHub App",
			wantAuthFlows: []minderv1.AuthorizationFlow{
				minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_GITHUB_APP_FLOW,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gh := &GitHub{
				propertyFetchers: properties.NewPropertyFetcherFactory(),
				providerClass:    tt.providerClass,
			}

			info := gh.ProviderClassInfo()
			require.NotNil(t, info)

			assert.Equal(t, tt.wantClass, info.Class)
			assert.Equal(t, tt.wantDisplay, info.DisplayName)
			assert.NotEmpty(t, info.Description)
			assert.Equal(t, providerDocsURL, info.DocumentationUrl)

			assert.ElementsMatch(t, tt.wantAuthFlows, info.SupportedAuthFlows)
			assert.ElementsMatch(t, wantEntities, info.SupportedEntities)

			for _, want := range wantTypes {
				assert.True(t, slices.Contains(info.SupportedProviderTypes, want),
					"expected SupportedProviderTypes to contain %v", want)
			}
		})
	}
}
