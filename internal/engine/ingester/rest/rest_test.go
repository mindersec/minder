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

package rest

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
	provifv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

func TestNewRestRuleDataIngest(t *testing.T) {
	t.Parallel()

	type args struct {
		restCfg *pb.RestType
		pbuild  *providers.ProviderBuilder
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid rest",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
				},
				pbuild: providers.NewProviderBuilder(
					&db.Provider{
						Name:    "osv",
						Version: provifv1.V1,
						Implements: []db.ProviderType{
							"rest",
						},
						Definition: json.RawMessage(`{
	"rest": {
		"base_url": "https://api.github.com/"
	}
}`),
					},
					db.ProviderAccessToken{},
					"token",
				),
			},
			wantErr: false,
		},
		{
			name: "invalid template",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "{{",
				},
				pbuild: providers.NewProviderBuilder(
					&db.Provider{
						Name:    "osv",
						Version: provifv1.V1,
						Implements: []db.ProviderType{
							"rest",
						},
						Definition: json.RawMessage(`{
	"rest": {
		"base_url": "https://api.github.com/"
	}
}`),
					},
					db.ProviderAccessToken{},
					"token",
				),
			},
			wantErr: true,
		},
		{
			name: "empty endpoint",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "",
				},
				pbuild: providers.NewProviderBuilder(
					&db.Provider{
						Name:    "osv",
						Version: provifv1.V1,
						Implements: []db.ProviderType{
							"rest",
						},
						Definition: json.RawMessage(`{
	"rest": {
		"endpoint": "https://api.github.com/"
	}
}`),
					},
					db.ProviderAccessToken{},
					"token",
				),
			},
			wantErr: true,
		},
		{
			name: "missing provider definition",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "",
				},
				pbuild: providers.NewProviderBuilder(
					&db.Provider{
						Name:    "osv",
						Version: provifv1.V1,
						Implements: []db.ProviderType{
							"rest",
						},
					},
					db.ProviderAccessToken{},
					"token",
				),
			},
			wantErr: true,
		},
		{
			name: "wrong provider definition",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "",
				},
				pbuild: providers.NewProviderBuilder(
					&db.Provider{
						Name:    "osv",
						Version: provifv1.V1,
						Implements: []db.ProviderType{
							"rest",
						},
						Definition: json.RawMessage(`{
	"rest": {
		"wrong": "https://api.github.com/"
	}
}`),
					},
					db.ProviderAccessToken{},
					"token",
				),
			},
			wantErr: true,
		},
		{
			name: "invalid provider definition",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "",
				},
				pbuild: providers.NewProviderBuilder(
					&db.Provider{
						Name:    "osv",
						Version: provifv1.V1,
						Implements: []db.ProviderType{
							"rest",
						},
						Definition: json.RawMessage(`{
	"rest": {
		"base_url": "https://api.github.com/"
}`),
					},
					db.ProviderAccessToken{},
					"token",
				),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewRestRuleDataIngest(tt.args.restCfg, tt.args.pbuild)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, got, "expected nil")
				return
			}

			require.NoError(t, err, "unexpected error")
			require.NotNil(t, got, "expected non-nil")
		})
	}
}
