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

package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/release-utils/tar"
)

// importLayouts is a utility function that reads OCI layouts from a
// directory and pushed them to the test registry. The directory name
// is used as the tat
func pushImageLayout(t *testing.T, layoutPath, tagPath string) {
	t.Helper()

	tag := fmt.Sprintf("%s/%s", os.Getenv("MINDER_TEST_REGISTRY"), tagPath)

	ref, err := name.ParseReference(tag)
	require.NoError(t, err)

	l, err := layout.ImageIndexFromPath(layoutPath)
	require.NoError(t, err)

	m, err := l.IndexManifest()
	require.NoError(t, err)

	desc := m.Manifests[0]
	require.True(t, desc.MediaType.IsImage(), "layout must be an image")

	i, err := l.Image(desc.Digest)
	require.NoError(t, err)

	// Push the image
	require.NoError(t, remote.Write(ref, i))
}

func TestBundleFromOCIImage(t *testing.T) {
	t.Parallel()
	if os.Getenv("MINDER_TEST_REGISTRY") == "" {
		t.Log("no test registry available, skipping")
		t.SkipNow()
	}

	// Extract the test images
	dir := t.TempDir()
	require.NoError(t, tar.Extract("testdata/images.tar.gz", dir))

	for _, tc := range []struct {
		name    string
		prepare func(t *testing.T, tag string)
		err     error
	}{
		{
			name: "normal-signed",
			prepare: func(t *testing.T, tag string) {
				t.Helper()
				pushImageLayout(t, filepath.Join(dir, "signed"), tag)
				pushImageLayout(t, filepath.Join(dir, "signed.sig"), tag+":sha256-992470149ecab5c4f1363b1943cc935dd58e16d833bbab3019949ff77d8b3060.sig")
			},
		},
		{
			name: "no-signature",
			prepare: func(t *testing.T, tag string) {
				t.Helper()
				pushImageLayout(t, filepath.Join(dir, "signed"), tag)
			},
			err: ErrProvenanceNotFoundOrIncomplete,
		},
		{
			name: "invalid-signature",
			prepare: func(t *testing.T, tag string) {
				t.Helper()
				pushImageLayout(t, filepath.Join(dir, "signed"), tag)
				pushImageLayout(t, filepath.Join(dir, "signed"), tag+":sha256-992470149ecab5c4f1363b1943cc935dd58e16d833bbab3019949ff77d8b3060.sig")
			},
			err: ErrProvenanceNotFoundOrIncomplete,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.prepare(t, fmt.Sprintf("bundle-from-oci-image/%s", tc.name))

			bundle, err := bundleFromOCIImage(
				context.Background(),
				fmt.Sprintf("%s/bundle-from-oci-image/%s", os.Getenv("MINDER_TEST_REGISTRY"), tc.name),
				authn.Anonymous,
			)

			if tc.err != nil {
				require.Error(t, err)
				require.True(t, errors.Is(err, tc.err))
				return
			}

			require.NoError(t, err)
			require.Len(t, bundle, 1)
		})
	}
}
