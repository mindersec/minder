// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestCatalogValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		catalog          Catalog
		expectErr        bool
		expectedProfiles int
		expectedFirst    string
	}{
		{
			name: "empty catalog",
			catalog: Catalog{
				RuleTypes: []*minderv1.RuleType{},
				Profiles:  []*minderv1.Profile{},
			},
			expectErr:        true,
			expectedProfiles: 0,
		},
		{
			name: "valid catalog",
			catalog: Catalog{
				RuleTypes: []*minderv1.RuleType{
					{Name: "test-rule"},
				},
				Profiles: []*minderv1.Profile{
					{
						Name: "valid",
						Repository: []*minderv1.Profile_Rule{
							{Type: "test-rule"},
						},
					},
				},
			},
			expectErr:        false,
			expectedProfiles: 1,
			expectedFirst:    "valid",
		},
		{
			name: "mixed valid and missing rule references",
			catalog: Catalog{
				RuleTypes: []*minderv1.RuleType{
					{Name: "test-rule"},
				},
				Profiles: []*minderv1.Profile{
					{
						Name: "valid",
						Repository: []*minderv1.Profile_Rule{
							{Type: "test-rule"},
						},
					},
					{
						Name: "invalid",
						Repository: []*minderv1.Profile_Rule{
							{Type: "missing-rule"},
						},
					},
				},
			},
			expectErr:        false,
			expectedProfiles: 1,
			expectedFirst:    "valid",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.catalog.Validate(func(string, ...any) {})

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, tt.catalog.Profiles, tt.expectedProfiles)
			if tt.expectedFirst != "" {
				require.Equal(t, tt.expectedFirst, tt.catalog.Profiles[0].Name)
			}
		})
	}
}
