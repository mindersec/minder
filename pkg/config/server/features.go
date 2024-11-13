// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"

	"github.com/mindersec/minder/internal/auth/jwt"
)

// FeaturesConfig is the configuration for the features
type FeaturesConfig struct {
	RoleFeatureMapping map[string]string `mapstructure:"role_feature_mapping"`
}

// GetFeaturesForRoles returns the features associated with the roles in the context
func (fc *FeaturesConfig) GetFeaturesForRoles(ctx context.Context) ([]string, error) {
	roles, err := extractRolesFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting roles: %v", err)
	}

	var features []string
	for _, role := range roles {
		if feature, ok := fc.RoleFeatureMapping[role]; ok {
			features = append(features, feature)
		}
	}
	return features, nil
}

// extractRolesFromContext extracts roles from the JWT in the context
func extractRolesFromContext(ctx context.Context) ([]string, error) {
	var realmAccess map[string]interface{}
	if claim, ok := jwt.GetUserClaimFromContext[map[string]interface{}](ctx, "realm_access"); ok {
		realmAccess = claim
	} else {
		return nil, fmt.Errorf("realm_access claim not found")
	}

	var roles []string
	if rolesInterface, ok := realmAccess["roles"].([]interface{}); ok {
		for _, role := range rolesInterface {
			if roleStr, ok := role.(string); ok {
				roles = append(roles, roleStr)
			}
		}
	} else {
		return nil, fmt.Errorf("roles not found in realm_access")
	}

	return roles, nil
}
