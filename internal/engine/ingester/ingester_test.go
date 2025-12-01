// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ingester provides necessary interfaces and implementations for ingesting
// data for rules.
package ingester

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/engine/ingester/artifact"
	"github.com/mindersec/minder/internal/engine/ingester/builtin"
	"github.com/mindersec/minder/internal/engine/ingester/git"
	"github.com/mindersec/minder/internal/engine/ingester/rest"
	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/providers/github/properties"
	"github.com/mindersec/minder/internal/providers/ratecache"
	"github.com/mindersec/minder/internal/providers/telemetry"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewRuleDataIngest(t *testing.T) {
	t.Parallel()

	type args struct {
		rt *pb.RuleType
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "artifact",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Ingest: &pb.RuleType_Definition_Ingest{
							Type:     artifact.ArtifactRuleDataIngestType,
							Artifact: &pb.ArtifactType{},
						},
					},
				},
			},
		},
		{
			name: "artifact missing",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Ingest: &pb.RuleType_Definition_Ingest{
							Type: artifact.ArtifactRuleDataIngestType,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "rest",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Ingest: &pb.RuleType_Definition_Ingest{
							Type: rest.RestRuleDataIngestType,
							Rest: &pb.RestType{
								Endpoint: "https://api.github.com/repos/Foo/Bar",
							},
						},
					},
				},
			},
		},
		{
			name: "rest missing",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Ingest: &pb.RuleType_Definition_Ingest{
							Type: rest.RestRuleDataIngestType,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "builtin",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Ingest: &pb.RuleType_Definition_Ingest{
							Type:    builtin.BuiltinRuleDataIngestType,
							Builtin: &pb.BuiltinType{},
						},
					},
				},
			},
		},
		{
			name: "builtin missing",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Ingest: &pb.RuleType_Definition_Ingest{
							Type: builtin.BuiltinRuleDataIngestType,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "git",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Ingest: &pb.RuleType_Definition_Ingest{
							Type: git.GitRuleDataIngestType,
							Git: &pb.GitType{
								CloneUrl: "https://github.com/staklok/minder.git",
							},
						},
					},
				},
			},
		},
		{
			name: "unsupported",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Ingest: &pb.RuleType_Definition_Ingest{
							Type: "unsupported",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := clients.NewRestClient(
				&pb.GitHubProviderConfig{},
				nil,
				nil,
				&ratecache.NoopRestClientCache{},
				credentials.NewGitHubTokenCredential("token"),
				clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
				properties.NewPropertyFetcherFactory(),
				"",
			)
			require.NoError(t, err)

			ingester, err := NewRuleDataIngest(tt.args.rt, client)
			if tt.wantErr {
				require.Error(t, err, "Expected error")
				require.Nil(t, ingester, "Expected nil")
				return
			}

			require.NoError(t, err, "Unexpected error")
			require.NotNil(t, ingester, "Expected non-nil")
		})
	}
}
