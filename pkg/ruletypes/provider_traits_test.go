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
		Traits        []pb.ProviderType
		ExpectedError string
	}{
		{
			Name:   "empty traits list passes",
			Traits: nil,
		},
		{
			Name:   "single valid trait passes",
			Traits: []pb.ProviderType{pb.ProviderType_PROVIDER_TYPE_GITHUB},
		},
		{
			Name: "multiple valid traits pass",
			Traits: []pb.ProviderType{
				pb.ProviderType_PROVIDER_TYPE_GIT,
				pb.ProviderType_PROVIDER_TYPE_GITHUB,
			},
		},
		{
			Name:          "unspecified trait fails",
			Traits:        []pb.ProviderType{pb.ProviderType_PROVIDER_TYPE_UNSPECIFIED},
			ExpectedError: "invalid provider_traits",
		},
		{
			Name:          "unspecified trait among valid ones fails",
			Traits:        []pb.ProviderType{pb.ProviderType_PROVIDER_TYPE_GITHUB, pb.ProviderType_PROVIDER_TYPE_UNSPECIFIED},
			ExpectedError: "invalid provider_traits",
		},
		{
			Name:          "unknown trait value fails",
			Traits:        []pb.ProviderType{pb.ProviderType(9999)},
			ExpectedError: "invalid provider_traits",
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
