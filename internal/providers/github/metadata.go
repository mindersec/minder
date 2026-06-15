// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"fmt"

	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	providerDocsBaseURL = "https://docs.mindersec.dev"
	providerDocsURL     = providerDocsBaseURL + "/integrations/provider_integrations/github"
)

// OAuthAuthorizationFlows is the list of authorization flows that the GitHub OAuth provider supports.
var OAuthAuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowUserInput,
	db.AuthorizationFlowOauth2AuthorizationCodeFlow,
}

// AppAuthorizationFlows is the list of authorization flows that the GitHub App provider supports.
var AppAuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowGithubAppFlow,
}

// ProviderClassInfo implements the Provider interface.
func (c *GitHub) ProviderClassInfo() *minderv1.ProviderClassInfo {
	supportedTypes := provifv1.ProviderTypesFromImpl(c)
	supportedEntities := []minderv1.Entity{
		minderv1.Entity_ENTITY_REPOSITORIES,
		minderv1.Entity_ENTITY_PULL_REQUESTS,
		minderv1.Entity_ENTITY_ARTIFACTS,
		minderv1.Entity_ENTITY_RELEASE,
	}
	//nolint:exhaustive
	switch c.providerClass {
	case db.ProviderClassGithub:
		return &minderv1.ProviderClassInfo{
			Class:                  string(db.ProviderClassGithub),
			DisplayName:            "GitHub OAuth",
			Description:            "GitHub provider using OAuth credentials.",
			SupportedProviderTypes: supportedTypes,
			SupportedAuthFlows:     dbAuthFlowsToPB(OAuthAuthorizationFlows),
			SupportedEntities:      supportedEntities,
			DocumentationUrl:       providerDocsURL,
		}
	case db.ProviderClassGithubApp:
		return &minderv1.ProviderClassInfo{
			Class:                  string(db.ProviderClassGithubApp),
			DisplayName:            "GitHub App",
			Description:            "GitHub App-based provider with installation-based authorization.",
			SupportedProviderTypes: supportedTypes,
			SupportedAuthFlows:     dbAuthFlowsToPB(AppAuthorizationFlows),
			SupportedEntities:      supportedEntities,
			DocumentationUrl:       providerDocsURL,
		}
	default:
		return nil
	}
}

// ProviderClassInfo returns class metadata for GitHub-backed provider classes.
// It uses a nil-pointer receiver to avoid needing a live client instance.
func ProviderClassInfo(class db.ProviderClass) (*minderv1.ProviderClassInfo, error) {
	info := (&GitHub{providerClass: class}).ProviderClassInfo()
	if info == nil {
		return nil, fmt.Errorf("unsupported GitHub provider class: %s", class)
	}
	return info, nil
}

func dbAuthFlowsToPB(flows []db.AuthorizationFlow) []minderv1.AuthorizationFlow {
	out := make([]minderv1.AuthorizationFlow, 0, len(flows))
	for _, f := range flows {
		switch f {
		case db.AuthorizationFlowNone:
			out = append(out, minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_NONE)
		case db.AuthorizationFlowUserInput:
			out = append(out, minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT)
		case db.AuthorizationFlowOauth2AuthorizationCodeFlow:
			out = append(out, minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW)
		case db.AuthorizationFlowGithubAppFlow:
			out = append(out, minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_GITHUB_APP_FLOW)
		}
	}

	return out
}
