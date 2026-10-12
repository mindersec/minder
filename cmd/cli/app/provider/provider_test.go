// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestGetImplementsAsStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider *minderv1.Provider
		want     []string
	}{
		{
			name:     "nil provider",
			provider: nil,
			want:     nil,
		},
		{
			name:     "empty implements",
			provider: &minderv1.Provider{},
			want:     nil,
		},
		{
			name: "valid implements",
			provider: &minderv1.Provider{
				Implements: []minderv1.ProviderType{
					minderv1.ProviderType_PROVIDER_TYPE_GITHUB,
					minderv1.ProviderType_PROVIDER_TYPE_GIT,
				},
			},
			want: []string{"github", "git"},
		},
		{
			name: "unspecified implement is ignored",
			provider: &minderv1.Provider{
				Implements: []minderv1.ProviderType{
					minderv1.ProviderType_PROVIDER_TYPE_UNSPECIFIED,
					minderv1.ProviderType_PROVIDER_TYPE_GITHUB,
				},
			},
			want: []string{"github"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getImplementsAsStrings(tt.provider)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAuthFlowsAsStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider *minderv1.Provider
		want     []string
	}{
		{
			name:     "nil provider",
			provider: nil,
			want:     nil,
		},
		{
			name:     "empty auth flows",
			provider: &minderv1.Provider{},
			want:     nil,
		},
		{
			name: "valid auth flows",
			provider: &minderv1.Provider{
				AuthFlows: []minderv1.AuthorizationFlow{
					minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
					minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW,
				},
			},
			want: []string{"user_input", "oauth2_authorization_code_flow"},
		},
		{
			name: "unspecified flow is ignored",
			provider: &minderv1.Provider{
				AuthFlows: []minderv1.AuthorizationFlow{
					minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_UNSPECIFIED,
					minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
				},
			},
			want: []string{"user_input"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getAuthFlowsAsStrings(tt.provider)
			assert.Equal(t, tt.want, got)
		})
	}
}
