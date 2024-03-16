//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mindpak

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestManifestWrite(t *testing.T) {
	t.Parallel()
	now := time.Unix(1709866805, 0)
	for _, tc := range []struct {
		name     string
		manifest *Manifest
		mustErr  bool
	}{
		{
			"normal",
			&Manifest{
				Metadata: &Metadata{
					Name:      "test",
					Namespace: "testspace",
					Version:   "v1.2.0",
					Date:      &now,
				},
				Files: &Files{
					Profiles: []*File{
						{
							Name: "profile.yaml",
							Hashes: map[HashAlgorithm]string{
								SHA256: "8b438ca800dfa20c6ca66ed83f05ef874cc1e1859d1a0a193b4c0727e5629977",
							},
						},
					},
					RuleTypes: []*File{
						{
							Name: "rule_type.yaml",
							Hashes: map[HashAlgorithm]string{
								SHA256: "0aecaf4d7ce19dc39679952c6951005e1396a5e615289ff3deb351873957d055",
							},
						},
					},
				},
			},
			false,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := bytes.NewBuffer([]byte{})
			err := tc.manifest.Write(b)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			man := &Manifest{}
			require.NoError(t, json.Unmarshal(b.Bytes(), man))

			if diff := cmp.Diff(tc.manifest, man, protocmp.Transform()); diff != "" {
				t.Fatalf("assertion failed: values are not equal\n--- expected\n+++ actual\n%v", diff)
			}
		})
	}

}
