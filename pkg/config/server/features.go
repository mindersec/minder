// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"

	"github.com/mindersec/minder/internal/auth/jwt"
)

// FeaturesConfig is the configuration for the features
type FeaturesConfig struct {
	// MembershipFeatureMapping maps a membership to a feature
	MembershipFeatureMapping map[string]string `mapstructure:"membership_feature_mapping"`
}

// GetFeaturesForMemberships returns the features associated with the memberships in the context
func (fc *FeaturesConfig) GetFeaturesForMemberships(ctx context.Context) []string {
	memberships := extractMembershipsFromContext(ctx)

	features := make([]string, 0, len(memberships))
	for _, m := range memberships {
		if feature := fc.MembershipFeatureMapping[m]; feature != "" {
			features = append(features, feature)
		}
	}

	return features
}

// extractMembershipsFromContext extracts memberships from the JWT in the context.
// Returns empty slice if no memberships are found.
func extractMembershipsFromContext(ctx context.Context) []string {
	realmAccess, ok := jwt.GetUserClaimFromContext[map[string]any](ctx, "realm_access")
	if !ok {
		return nil
	}

	rawMemberships, ok := realmAccess["roles"].([]any)
	if !ok {
		return nil
	}

	memberships := make([]string, 0, len(rawMemberships))
	for _, membership := range rawMemberships {
		if membershipStr, ok := membership.(string); ok {
			memberships = append(memberships, membershipStr)
		}
	}

	return memberships
}
