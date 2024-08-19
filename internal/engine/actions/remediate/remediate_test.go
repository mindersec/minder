// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

// Package remediate_test provides tests for the remediate package.
package remediate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/stacklok/minder/internal/engine/actions/remediate"
	"github.com/stacklok/minder/internal/engine/actions/remediate/noop"
	"github.com/stacklok/minder/internal/engine/actions/remediate/rest"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/providers/testproviders"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var err error
			var provider provifv1.Provider
			if tt.provider != nil {
				provider, err = tt.provider()
				require.NoError(t, err)
			}
			result, err := remediate.NewRuleRemediator(tt.ruleType, provider)
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
