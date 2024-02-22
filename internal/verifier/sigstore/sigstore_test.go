// Copyright 2024 Stacklok, Inc
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

package sigstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadTrustedRoot(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		rootSource string
		prepare    func(*testing.T, string) string
		mustErr    bool
	}{
		{
			name:       "default root (blank)",
			rootSource: "",
		},
		{
			name:       "default root (constant)",
			rootSource: SigstorePublicTrustedRootRepo,
		},
		{
			name:       "invalid repo",
			rootSource: "example.com",
			mustErr:    true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmp := t.TempDir()
			trustedRoot, err := readTrustedRoot(tc.rootSource, tmp)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, trustedRoot)
		})
	}
}

func TestRootJson(t *testing.T) {
	sampleData := []byte("test")
	rootSource := "testrepo.com"
	t.Parallel()
	for _, tc := range []struct {
		name    string
		repo    string
		prepare func(*testing.T, string)
		expect  []byte
		mustErr bool
	}{
		{
			name:    "blank root",
			repo:    "",
			prepare: func(_ *testing.T, _ string) {},
			expect:  nil,
			mustErr: false,
		},
		{
			name: "normal",
			prepare: func(t *testing.T, s string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(s, rootSource), os.FileMode(0o700)))
				require.NoError(t, os.WriteFile(filepath.Join(s, rootSource, rootTUFPath), sampleData, os.FileMode(0o600)))
			},
			expect:  sampleData,
			mustErr: false,
		},
		{
			name:    "no cache match",
			repo:    rootSource,
			prepare: func(_ *testing.T, _ string) {},
			expect:  nil,
			mustErr: false,
		},
		{
			name: "error reading",
			repo: rootSource,
			prepare: func(t *testing.T, s string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(s, rootSource, rootTUFPath), os.FileMode(0o700)))
			},
			expect:  nil,
			mustErr: true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cacheDir := t.TempDir()
			tc.prepare(t, cacheDir)
			res, err := readRootJson(rootSource, cacheDir)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, tc.expect, res)
		})
	}
}
