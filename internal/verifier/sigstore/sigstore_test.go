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
	"net/url"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/stretchr/testify/require"
)

func TestGetSigstoreOptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                 string
		rootSource           string
		prepare              func(*testing.T, string) string
		mustErr              bool
		expectedTUFOptions   *tuf.Options
		expectedVerifierOpts []verify.VerifierOption
	}{
		{
			name:       "default root (blank)",
			rootSource: "",
			expectedTUFOptions: &tuf.Options{
				RepositoryBaseURL: SigstorePublicTrustedRootRepo,
				DisableLocalCache: true,
			},
			expectedVerifierOpts: []verify.VerifierOption{
				verify.WithSignedCertificateTimestamps(1),
				verify.WithTransparencyLog(1),
				verify.WithObserverTimestamps(1),
			},
		},
		{
			name:       "sigstore's PGI root",
			rootSource: SigstorePublicTrustedRootRepo,
			expectedTUFOptions: &tuf.Options{
				RepositoryBaseURL: SigstorePublicTrustedRootRepo,
				DisableLocalCache: true,
			},
			expectedVerifierOpts: []verify.VerifierOption{
				verify.WithSignedCertificateTimestamps(1),
				verify.WithTransparencyLog(1),
				verify.WithObserverTimestamps(1),
			},
		},
		{
			name:       "GitHub's Sigstore root",
			rootSource: GitHubSigstoreTrustedRootRepo,
			expectedTUFOptions: &tuf.Options{
				RepositoryBaseURL: GitHubSigstoreTrustedRootRepo,
				DisableLocalCache: true,
			},
			expectedVerifierOpts: []verify.VerifierOption{
				verify.WithObserverTimestamps(1),
			},
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
			tuf, verifier, err := getSigstoreOptions(tc.rootSource)
			if tc.mustErr {
				require.Error(t, err)
				require.Nil(t, tuf)
				require.Nil(t, verifier)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, tuf)
			require.NotNil(t, verifier)
			// Verify the TUF options
			require.Equal(t, tc.expectedTUFOptions.DisableLocalCache, tuf.DisableLocalCache)
			tufURL, err := url.Parse(tuf.RepositoryBaseURL)
			require.NoError(t, err)
			require.Equal(t, tc.expectedTUFOptions.RepositoryBaseURL, tufURL.Hostname())
			// Verify the verifier options - checks the number of options
			require.Equal(t, len(tc.expectedVerifierOpts), len(verifier))

		})
	}
}
