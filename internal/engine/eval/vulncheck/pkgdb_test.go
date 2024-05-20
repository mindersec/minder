// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNpmPkgDb(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockHandler    http.HandlerFunc
		depName        string
		patchedVersion string
		expectError    bool
		errorToExpect  error
		expectReply    *packageJson
	}{
		{
			name: "ValidResponseLatest",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/my-package/latest", r.URL.Path, "unexpected path")
				data := packageJson{
					Name:    "my-package",
					Version: "1.0.0",
					Dist: struct {
						Integrity string `json:"integrity"`
						Tarball   string `json:"tarball"`
					}{
						Integrity: "sha512-...",
						Tarball:   "https://example.com/path/to/tarball.tgz",
					},
				}
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
			depName: "my-package",
			expectReply: &packageJson{
				Name:    "my-package",
				Version: "1.0.0",
				Dist: struct {
					Integrity string `json:"integrity"`
					Tarball   string `json:"tarball"`
				}{
					Integrity: "sha512-...",
					Tarball:   "https://example.com/path/to/tarball.tgz",
				},
			},
		},
		{
			name: "ValidResponseVersioned",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/my-package/1.0.1", r.URL.Path, "unexpected path")
				data := packageJson{
					Name:    "my-package",
					Version: "1.0.1",
					Dist: struct {
						Integrity string `json:"integrity"`
						Tarball   string `json:"tarball"`
					}{
						Integrity: "sha512-...",
						Tarball:   "https://example.com/path/to/tarball.tgz",
					},
				}
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:        "my-package",
			patchedVersion: "1.0.1",
			expectReply: &packageJson{
				Name:    "my-package",
				Version: "1.0.1",
				Dist: struct {
					Integrity string `json:"integrity"`
					Tarball   string `json:"tarball"`
				}{
					Integrity: "sha512-...",
					Tarball:   "https://example.com/path/to/tarball.tgz",
				},
			},
		},
		{
			name: "Non200Response",
			mockHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte("Not Found"))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:       "non-existing-package",
			expectError:   true,
			errorToExpect: ErrPkgNotFound,
		},
		{
			name: "InvalidJSON",
			mockHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("{ invalid json }"))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:     "package-with-invalid-json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.mockHandler)
			defer server.Close()

			repo := newNpmRepository(server.URL)

			dep := &pb.Dependency{
				Name: tt.depName,
			}

			var latest bool
			if tt.patchedVersion == "" {
				latest = true
			}
			reply, err := repo.SendRecvRequest(context.Background(), dep, tt.patchedVersion, latest)
			if tt.expectError {
				if tt.errorToExpect != nil {
					assert.ErrorIs(t, err, tt.errorToExpect, "Expected error")
				}
				assert.Error(t, err, "Expected error")
			} else {
				assert.NoError(t, err, "Expected no error")
				require.Equal(t, tt.expectReply.IndentedString(0, "", nil), reply.IndentedString(0, "", nil), "expected reply to match mock data")
			}
		})
	}
}

func TestPackageJsonLineHasDependency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		versionPack string
		pkgJson     *packageJson
		want        bool
	}{
		{
			name:        "empty versionPack",
			versionPack: "",
			pkgJson:     &packageJson{Name: "@next/swc-linux-arm64-gnu"},
			want:        false,
		},
		{
			name: "versionPack with containing a mismatching package name but matching versions",
			versionPack: `"node_modules/@next/swc-linux-arm64-gnu": {
  "version": "13.4.18",
13.4.18`,
			pkgJson: &packageJson{Name: "@next/swc-linux-x64-musl"},
			want:    false,
		},
		{
			name: "versionPack with a matching package name but mismatching versions",
			versionPack: `"node_modules/@next/swc-linux-arm64-gnu": {
  "version": "13.4.18",
13.4.20`,
			pkgJson: &packageJson{Name: "@next/swc-linux-arm64-gnu"},
			want:    false,
		},
		{
			name: "versionPack with matching package name and version",
			versionPack: `"node_modules/@next/swc-linux-arm64-gnu": {
  "version": "13.4.18",
13.4.18`,
			pkgJson: &packageJson{Name: "@next/swc-linux-arm64-gnu"},
			want:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.pkgJson.LineHasDependency(tt.versionPack); got != tt.want {
				t.Errorf("packageJson.LineHasDependency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPyPiReplyLineHasDependency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		line         string
		reply        *PyPiReply
		expectRetval bool
	}{
		{
			name: "Match, equal version",
			line: "requests==2.19.0",
			reply: &PyPiReply{
				Info: struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "requests",
					Version: "3.4.5",
				},
			},
			expectRetval: true,
		},
		{
			name: "Match, less or equal version",
			line: "requests<=2.19.0",
			reply: &PyPiReply{
				Info: struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "requests",
					Version: "3.4.5",
				},
			},
			expectRetval: true,
		},
		{
			name: "Match, greater or equal version",
			line: "requests>=2.19.0",
			reply: &PyPiReply{
				Info: struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "requests",
					Version: "3.4.5",
				},
			},
			expectRetval: true,
		},
		{
			name: "Not a match greater or equal version",
			line: "otherpackage>=2.19.0",
			reply: &PyPiReply{
				Info: struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "requests",
					Version: "3.4.5",
				},
			},
			expectRetval: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
		})

		require.Equal(t, tt.expectRetval, tt.reply.LineHasDependency(tt.line), "expected reply to match mock data")
	}
}

func TestPyPiPkgDb(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		mockPyPiHandler http.HandlerFunc
		depName         string
		patchedVersion  string
		expectError     bool
		errorToExpect   error
		expectReply     string
	}{
		{
			name: "ValidResponseLatest",
			mockPyPiHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/requests/json", r.URL.Path, "unexpected path")
				data := PyPiReply{}
				data.Info.Name = "requests"
				data.Info.Version = "2.25.1"

				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:     "requests",
			expectReply: "requests>=2.25.1",
		},
		{
			name: "ValidResponseVersioned",
			mockPyPiHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/requests/2.25.0/json", r.URL.Path, "unexpected path")
				data := PyPiReply{}
				data.Info.Name = "requests"
				data.Info.Version = "2.25.0"

				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:        "requests",
			patchedVersion: "2.25.0",
			expectReply:    "requests>=2.25.0",
		},
		{
			name: "Non200Response",
			mockPyPiHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte("Not Found"))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:       "non-existing-package",
			expectError:   true,
			errorToExpect: ErrPkgNotFound,
		},
		{
			name: "InvalidJSON",
			mockPyPiHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("{ invalid json }"))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:     "package-with-invalid-json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pyPiMockServer := httptest.NewServer(tt.mockPyPiHandler)
			defer pyPiMockServer.Close()

			repo := newPyPIRepository(pyPiMockServer.URL)
			assert.NotNil(t, repo, "Failed to create repository")

			dep := &pb.Dependency{
				Name: tt.depName,
			}

			var latest bool
			if tt.patchedVersion == "" {
				latest = true
			}
			reply, err := repo.SendRecvRequest(context.Background(), dep, tt.patchedVersion, latest)
			if tt.expectError {
				if tt.errorToExpect != nil {
					assert.ErrorIs(t, err, tt.errorToExpect, "Expected error")
				}
				assert.Error(t, err, "Expected error")
			} else {
				assert.NoError(t, err, "Expected no error")
				actualReply := reply.IndentedString(0,
					"requests>=2.19.0",
					&pb.Dependency{
						Name:    "requests",
						Version: "2.19.0",
					})
				require.Equal(t, tt.expectReply, actualReply, "expected reply to match mock data")
			}
		})
	}
}

func TestGoPkgDb(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		mockProxyHandler http.HandlerFunc
		mockSumHandler   http.HandlerFunc
		depName          string
		patchedVersion   string
		expectError      bool
		errorToExpect    error
		expectReply      *goModPackage
	}{
		{
			name: "ValidResponseLatest",
			mockProxyHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/golang.org/x/text/@latest", r.URL.Path, "unexpected path")
				data := goModPackage{
					Name:    "golang.org/x/text",
					Version: "v0.13.0",
				}
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
			mockSumHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`19383665
golang.org/x/text v0.13.0 h1:ablQoSUd0tRdKxZewP80B+BaqeKJuVhuRxj/dkrun3k=
golang.org/x/text v0.13.0/go.mod h1:TvPlkZtksWOMsz7fbANvkp4WM8x/WCo/om8BMLbz+aE=

go.sum database tree
19777102
+g50vJoV4VVGa6aiQF3LYGUZHEP4pvGkW38vlR7WKtU=

— sum.golang.org Az3griMTLHyRzj7jMuyEt85a2JnegVME6Lx3xLEBdzTd3FMDiD5y3bHV24rcl0yijOtWxV0zyygwTdo/rnaennuoqgU=`))
				assert.NoError(t, err, "Failed to write mock response")
			},
			depName: "golang.org/x/text",
			expectReply: &goModPackage{
				Name:           "golang.org/x/text",
				Version:        "v0.13.0",
				ModuleHash:     "h1:ablQoSUd0tRdKxZewP80B+BaqeKJuVhuRxj/dkrun3k=",
				DependencyHash: "h1:TvPlkZtksWOMsz7fbANvkp4WM8x/WCo/om8BMLbz+aE=",
			},
		},
		{
			name: "ValidResponseVersioned",
			mockProxyHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/golang.org/x/text/@v/v0.13.1.info", r.URL.Path, "unexpected path")
				data := goModPackage{
					Name:    "golang.org/x/text",
					Version: "v0.13.1",
				}
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
			mockSumHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`19383665
golang.org/x/text v0.13.1 h1:ablQoSUd0tRdKxZewP80B+BaqeKJuVhuRxj/dkrun3k=
golang.org/x/text v0.13.1/go.mod h1:TvPlkZtksWOMsz7fbANvkp4WM8x/WCo/om8BMLbz+aE=

go.sum database tree
19777102
+g50vJoV4VVGa6aiQF3LYGUZHEP4pvGkW38vlR7WKtU=

— sum.golang.org Az3griMTLHyRzj7jMuyEt85a2JnegVME6Lx3xLEBdzTd3FMDiD5y3bHV24rcl0yijOtWxV0zyygwTdo/rnaennuoqgU=`))
				assert.NoError(t, err, "Failed to write mock response")
			},
			depName:        "golang.org/x/text",
			patchedVersion: "0.13.1",
			expectReply: &goModPackage{
				Name:           "golang.org/x/text",
				Version:        "v0.13.1",
				ModuleHash:     "h1:ablQoSUd0tRdKxZewP80B+BaqeKJuVhuRxj/dkrun3k=",
				DependencyHash: "h1:TvPlkZtksWOMsz7fbANvkp4WM8x/WCo/om8BMLbz+aE=",
			},
		},
		{
			name: "Non200ResponseProxy",
			mockProxyHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte("Not Found"))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:       "non-existing-package",
			expectError:   true,
			errorToExpect: ErrPkgNotFound,
		},
		{
			name: "InvalidJSONProxy",
			mockProxyHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("{ invalid json }"))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:     "package-with-invalid-json",
			expectError: true,
		},
		{
			name: "MissingVersionProxy",
			mockProxyHandler: func(w http.ResponseWriter, _ *http.Request) {
				data := goModPackage{
					Name: "golang.org/x/text",
				}
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
				w.WriteHeader(http.StatusOK)
			},
			depName:     "package-with-invalid-json",
			expectError: true,
		},
		{
			name: "Non200ResponseSum",
			mockProxyHandler: func(w http.ResponseWriter, _ *http.Request) {
				data := goModPackage{
					Version: "v0.13.0",
				}
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
			mockSumHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			depName:       "golang.org/x/text",
			expectError:   true,
			errorToExpect: ErrPkgNotFound,
		},
		{
			name: "TooFewLinesSum",
			mockProxyHandler: func(w http.ResponseWriter, _ *http.Request) {
				data := goModPackage{
					Version: "v0.13.0",
				}
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(data)
				if err != nil {
					t.Fatal(err)
				}
			},
			mockSumHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`19383665
golang.org/x/text v0.13.0 h1:ablQoSUd0tRdKxZewP80B+BaqeKJuVhuRxj/dkrun3k=`))
				assert.NoError(t, err, "Failed to write mock response")
			},
			depName:     "golang.org/x/text",
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proxyServer := httptest.NewServer(tt.mockProxyHandler)
			defer proxyServer.Close()

			sumServer := httptest.NewServer(tt.mockSumHandler)
			defer proxyServer.Close()

			repo := newGoProxySumRepository(proxyServer.URL, sumServer.URL)
			assert.NotNil(t, repo, "Failed to create repository")

			dep := &pb.Dependency{
				Name: tt.depName,
			}

			var latest bool
			if tt.patchedVersion == "" {
				latest = true
			}
			reply, err := repo.SendRecvRequest(context.Background(), dep, tt.patchedVersion, latest)
			if tt.expectError {
				if tt.errorToExpect != nil {
					assert.ErrorIs(t, err, tt.errorToExpect, "Expected error")
				}
				assert.Error(t, err, "Expected error")
			} else {
				assert.NoError(t, err, "Expected no error")
				require.Equal(t, tt.expectReply.IndentedString(0, "", nil), reply.IndentedString(0, "", nil), "expected reply to match mock data")
			}
		})
	}
}

func TestRepoCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		ecoConfig      *ecosystemConfig
		expectError    bool
		expectSameRepo bool
	}{
		{
			name: "ValidEcosystemCachesConnections",
			ecoConfig: &ecosystemConfig{
				Name: "npm",
				PackageRepository: packageRepository{
					Url: "http://mock.url",
				},
			},
			expectError: false,
		},
		{
			name: "ErrorWithUnknownEcosystem",
			ecoConfig: &ecosystemConfig{
				Name: "unknown-ecosystem",
				PackageRepository: packageRepository{
					Url: "http://mock.url",
				},
			},
			expectError: true,
		},
	}

	cache := newRepoCache()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			newRepo, err := cache.newRepository(tt.ecoConfig)
			if tt.expectError {
				assert.Error(t, err, "Expected error")
				return
			}

			assert.NoError(t, err, "Failed to fetch repository for the first time")
			cachedRepo, err := cache.newRepository(tt.ecoConfig)
			assert.NoError(t, err, "Failed to fetch repository for the second time")
			assert.Equal(t, newRepo, cachedRepo, "Repositories from cache should be the same instance")
		})
	}
}
