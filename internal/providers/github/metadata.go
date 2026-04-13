// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"fmt"

	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	providerDocsBaseURL = "https://docs.mindersec.dev"
	providerDocsURL     = providerDocsBaseURL + "/integrations/provider_integrations/github"
)

// OAuthImplements is the list of provider types that the GitHub OAuth provider implements.
var OAuthImplements = []db.ProviderType{
	db.ProviderTypeGithub,
	db.ProviderTypeGit,
	db.ProviderTypeRest,
	db.ProviderTypeRepoLister,
}

// OAuthAuthorizationFlows is the list of authorization flows that the GitHub OAuth provider supports.
var OAuthAuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowUserInput,
	db.AuthorizationFlowOauth2AuthorizationCodeFlow,
}

// AppImplements is the list of provider types that the GitHub App provider implements.
var AppImplements = []db.ProviderType{
	db.ProviderTypeGithub,
	db.ProviderTypeGit,
	db.ProviderTypeRest,
	db.ProviderTypeRepoLister,
}

// AppAuthorizationFlows is the list of authorization flows that the GitHub App provider supports.
var AppAuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowGithubAppFlow,
}

// ProviderClassInfo returns class metadata for GitHub-backed provider classes.
func ProviderClassInfo(class db.ProviderClass) (*minderv1.ProviderClassInfo, error) {
	switch class {
	case db.ProviderClassGithub:
		return &minderv1.ProviderClassInfo{
			Class:                  string(db.ProviderClassGithub),
			DisplayName:            "GitHub OAuth",
			Description:            "GitHub provider using OAuth credentials.",
			SupportedProviderTypes: dbProviderTypesToPB(OAuthImplements),
			SupportedAuthFlows:     dbAuthFlowsToPB(OAuthAuthorizationFlows),
			SupportedEntities: []minderv1.Entity{
				minderv1.Entity_ENTITY_REPOSITORIES,
				minderv1.Entity_ENTITY_PULL_REQUESTS,
				minderv1.Entity_ENTITY_ARTIFACTS,
				minderv1.Entity_ENTITY_RELEASE,
			},
			DocumentationUrl: providerDocsURL,
		}, nil
	case db.ProviderClassGithubApp:
		return &minderv1.ProviderClassInfo{
			Class:                  string(db.ProviderClassGithubApp),
			DisplayName:            "GitHub App",
			Description:            "GitHub App-based provider with installation-based authorization.",
			SupportedProviderTypes: dbProviderTypesToPB(AppImplements),
			SupportedAuthFlows:     dbAuthFlowsToPB(AppAuthorizationFlows),
			SupportedEntities: []minderv1.Entity{
				minderv1.Entity_ENTITY_REPOSITORIES,
				minderv1.Entity_ENTITY_PULL_REQUESTS,
				minderv1.Entity_ENTITY_ARTIFACTS,
				minderv1.Entity_ENTITY_RELEASE,
			},
			DocumentationUrl: providerDocsURL,
		}, nil
	case db.ProviderClassGhcr:
		fallthrough
	case db.ProviderClassDockerhub:
		fallthrough
	case db.ProviderClassGitlab:
		return nil, fmt.Errorf("unsupported GitHub provider class: %s", class)
	default:
		return nil, fmt.Errorf("unsupported GitHub provider class: %s", class)
	}
}

// ProviderClassInfo implements the Provider interface.
func (c *GitHub) ProviderClassInfo() *minderv1.ProviderClassInfo {
	info, err := ProviderClassInfo(c.providerClass)
	if err != nil {
		return nil
	}

	return info
}

func dbProviderTypesToPB(types []db.ProviderType) []minderv1.ProviderType {
	out := make([]minderv1.ProviderType, 0, len(types))
	for _, t := range types {
		switch t {
		case db.ProviderTypeGit:
			out = append(out, minderv1.ProviderType_PROVIDER_TYPE_GIT)
		case db.ProviderTypeGithub:
			out = append(out, minderv1.ProviderType_PROVIDER_TYPE_GITHUB)
		case db.ProviderTypeRest:
			out = append(out, minderv1.ProviderType_PROVIDER_TYPE_REST)
		case db.ProviderTypeRepoLister:
			out = append(out, minderv1.ProviderType_PROVIDER_TYPE_REPO_LISTER)
		case db.ProviderTypeOci:
			out = append(out, minderv1.ProviderType_PROVIDER_TYPE_OCI)
		case db.ProviderTypeImageLister:
			out = append(out, minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER)
		}
	}

	return out
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
