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
	"io/fs"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestReadSource(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		path       string
		mustErr    bool
		filesMatch bool
		expect     *Bundle
	}{
		{
			"normal",
			"testdata/t1",
			false,
			true,
			&Bundle{
				Manifest: &Manifest{},
				Metadata: &Metadata{},
				Files: &Files{
					Profiles: []*File{
						{
							Name:   "branch-protection.yaml",
							Hashes: map[HashAlgorithm]string{SHA256: "51437d1e5049a16513b9cc9d6d93d6b25625f51e74e0861fba837cdf1d2b5f01"},
						},
					},
					RuleTypes: []*File{
						{
							Name:   "secret_scanning.yaml",
							Hashes: map[HashAlgorithm]string{SHA256: "fc3e782516d0de46e89610af0b0bab04783e0e6e875c6efa64c9dfb3ef127964"},
						},
					},
				},
				Source: nil,
			},
		},
		{
			"wrong-hash",
			"testdata/t1",
			false,
			false,
			&Bundle{
				Manifest: &Manifest{},
				Metadata: &Metadata{},
				Files: &Files{
					Profiles: []*File{
						{
							Name:   "branch-protection.yaml",
							Hashes: map[HashAlgorithm]string{SHA256: "AAf3682a1cb5ab92c0cc71dd913338bf40a89ec324024f8d3f500be0e2aa4a9ae1"},
						},
					},
					RuleTypes: []*File{
						{
							Name:   "secret_scanning.yaml",
							Hashes: map[HashAlgorithm]string{SHA256: "AA572089a9a490d1b7d07f2a1f6845ae1f18af27a6a13a605de7cef8a910427084"},
						},
					},
				},
				Source: nil,
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := Bundle{}
			m.Source = os.DirFS(tc.path).(fs.StatFS)
			err := m.ReadSource()
			if tc.mustErr {
				require.Error(t, err)
				return
			}

			diff := cmp.Diff(&tc.expect.Files, &m.Files, protocmp.Transform())
			if tc.filesMatch {
				require.Empty(t, diff, "file hashes don't match:\n%v", diff)
			} else {
				require.NotEmpty(t, diff, "file hashes should not match")
			}

		})

	}
}
