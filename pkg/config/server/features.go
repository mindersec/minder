// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"

	"github.com/mindersec/minder/internal/auth/jwt"
)

// FeaturesConfig is the configuration for the features
type FeaturesConfig struct {
	// RoleFeatureMapping maps a role to a feature
	RoleFeatureMapping map[string]string `mapstructure:"role_feature_mapping"`
}

// GetFeaturesForRoles returns the features associated with the roles in the context
func (fc *FeaturesConfig) GetFeaturesForRoles(ctx context.Context) []string {
	roles := extractRolesFromContext(ctx)

	var features []string
	for _, role := range roles {
		if feature, ok := fc.RoleFeatureMapping[role]; ok {
			features = append(features, feature)
		}
	}

	return features
}

// extractRolesFromContext extracts roles from the JWT in the context.
// Returns empty slice if no roles are found.
func extractRolesFromContext(ctx context.Context) []string {
	realmAccess, ok := jwt.GetUserClaimFromContext[map[string]interface{}](ctx, "realm_access")
	if !ok {
		return nil
	}

	rolesInterface, ok := realmAccess["roles"].([]interface{})
	if !ok {
		return nil
	}

	roles := make([]string, len(rolesInterface))
	for i, role := range rolesInterface {
		if roleStr, ok := role.(string); ok {
			roles[i] = roleStr
		}
	}

	return roles
}
