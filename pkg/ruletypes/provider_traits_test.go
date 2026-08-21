// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestValidateProviderTraits(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		Traits        []string
		ExpectedError string
	}{
		{
			Name:   "empty traits list passes",
			Traits: nil,
		},
		{
			Name:   "single valid trait passes",
			Traits: []string{"github"},
		},
		{
			Name:   "multiple valid traits pass",
			Traits: []string{"git", "github"},
		},
		{
			Name:          "unknown trait fails",
			Traits:        []string{"gihtub"},
			ExpectedError: `unknown trait "gihtub"`,
		},
		{
			Name:          "unknown trait among valid ones fails",
			Traits:        []string{"github", "gitlab"},
			ExpectedError: `unknown trait "gitlab"`,
		},
		{
			Name:          "numeric enum value is no longer accepted",
			Traits:        []string{"1"},
			ExpectedError: `unknown trait "1"`,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			ruleType := &pb.RuleType{
				Def: &pb.RuleType_Definition{
					ProviderTraits: scenario.Traits,
				},
			}

			err := validateProviderTraits(ruleType)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}
