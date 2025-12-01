// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package remediate_test provides tests for the remediate package.
package remediate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/engine/actions/remediate"
	"github.com/mindersec/minder/internal/engine/actions/remediate/noop"
	"github.com/mindersec/minder/internal/engine/actions/remediate/rest"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/telemetry"
	"github.com/mindersec/minder/internal/providers/testproviders"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

var (
	simpleBodyTemplate = "{\"foo\": \"bar\"}"
)

func TestNewRuleRemediator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ruleType  *pb.RuleType
		wantError bool
		wantType  engif.Action
		provider  func() (provifv1.Provider, error)
	}{
		{
			name: "Test Noop Remediate",
			ruleType: &pb.RuleType{
				Def: &pb.RuleType_Definition{}, // No remediate field set
			},
			wantError: false, // Expecting a NoopRemediate instance (or whichever condition you check for)
			wantType:  &noop.Remediator{},
		},
		{
			name: "Test REST Remediate",
			ruleType: &pb.RuleType{
				Def: &pb.RuleType_Definition{
					Remediate: &pb.RuleType_Definition_Remediate{
						Type: rest.RemediateType,
						Rest: &pb.RestType{
							Method:   "POST",
							Endpoint: "{{.Profile.endpoint}}",
							Body:     &simpleBodyTemplate,
						},
					},
				},
			},
			provider:  HTTPProvider,
			wantError: false, // Expecting a NoopRemediate instance (or whichever condition you check for)
			wantType:  &rest.Remediator{},
		},
		{
			name: "Test REST Remediate with wrong provider type",
			ruleType: &pb.RuleType{
				Def: &pb.RuleType_Definition{
					Remediate: &pb.RuleType_Definition_Remediate{
						Type: rest.RemediateType,
						Rest: &pb.RestType{
							Method:   "POST",
							Endpoint: "{{.Profile.endpoint}}",
							Body:     &simpleBodyTemplate,
						},
					},
				},
			},
			provider:  GitProvider,
			wantError: true,
		},
		{
			name: "Test Rest Remediate Without Config",
			ruleType: &pb.RuleType{
				Def: &pb.RuleType_Definition{
					Remediate: &pb.RuleType_Definition_Remediate{
						Type: rest.RemediateType,
					},
				},
			},
			provider:  HTTPProvider,
			wantError: true,
		},
		{
			name: "Test made up remediator",
			ruleType: &pb.RuleType{
				Def: &pb.RuleType_Definition{
					Remediate: &pb.RuleType_Definition_Remediate{
						Type: "madeup",
					},
				},
			},
			wantError: true,
		},
		// ... Add more test cases as needed
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var err error
			var provider provifv1.Provider
			if tt.provider != nil {
				provider, err = tt.provider()
				require.NoError(t, err)
			}
			result, err := remediate.NewRuleRemediator(
				tt.ruleType, provider, models.ActionOptOn)
			if tt.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.IsType(t, tt.wantType, result) // Or whichever condition you check for
		})
	}
}

func HTTPProvider() (provifv1.Provider, error) {
	cfg := pb.RESTProviderConfig{BaseUrl: proto.String("https://api.github.com/")}
	return testproviders.NewRESTProvider(
		&cfg,
		telemetry.NewNoopMetrics(),
		credentials.NewGitHubTokenCredential("token"),
	)
}

func GitProvider() (provifv1.Provider, error) {
	return testproviders.NewGitProvider(credentials.NewEmptyCredential()), nil
}
