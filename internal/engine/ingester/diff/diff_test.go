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

// Package diff provides the diff rule data ingest engine
package diff

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestGetEcosystemForFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		filename       string
		diffIngestCfg  *pb.DiffType
		expectedEcoSys DependencyEcosystem
	}{
		{
			name:     "Exact match",
			filename: "package-lock.json",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: "package-lock.json",
					},
				},
			},
			expectedEcoSys: DepEcosystemNPM,
		},
		{
			name:     "Wildcard match",
			filename: "/path/to/package-lock.json",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: fmt.Sprintf("%s%s", wildcard, "package-lock.json"),
					},
				},
			},
			expectedEcoSys: DepEcosystemNPM,
		},
		{
			name:     "Depfile without wildcard does not match subdirectory",
			filename: "/path/to/package-lock.json",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: "package-lock.json",
					},
				},
			},
			expectedEcoSys: DepEcosystemNPM,
		},
		{
			name:     "Wildcard not a match - wrong filename",
			filename: "/path/to/not-package-lock.json",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: fmt.Sprintf("%s/%s", wildcard, "package-lock.json"),
					},
				},
			},
			expectedEcoSys: DepEcosystemNone,
		},
		{
			name:     "No match",
			filename: "/path/to/README.md",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: fmt.Sprintf("%s%s", wildcard, "package-lock.json"),
					},
				},
			},
			expectedEcoSys: DepEcosystemNone,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			di := &Diff{
				cfg: tt.diffIngestCfg,
			}
			result := di.getEcosystemForFile(tt.filename)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedEcoSys, result)
		})
	}
}
