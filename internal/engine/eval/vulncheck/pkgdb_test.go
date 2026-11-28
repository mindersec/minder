// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

	pbinternal "github.com/mindersec/minder/internal/proto"
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.mockHandler)
			defer server.Close()

			repo := newNpmRepository(server.URL)

			dep := &pbinternal.Dependency{
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
			name: "Match, extras are not part of the name",
			line: "requests[security]==2.19.0",
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
			name: "Match, compatible release",
			line: "requests~=2.19",
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
			name: "No match, extras of another package",
			line: "requests-oauthlib[signedtoken]==1.3.1",
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
		{
			name: "Match, different case and underscores",
			line: "Some-package==1.1.0",
			reply: &PyPiReply{
				Info: struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "some_package",
					Version: "1.1.0",
				},
			},
			expectRetval: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expectRetval, tt.reply.LineHasDependency(tt.line), "expected reply to match mock data")
		})
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pyPiMockServer := httptest.NewServer(tt.mockPyPiHandler)
			defer pyPiMockServer.Close()

			repo := newPyPIRepository(pyPiMockServer.URL)
			assert.NotNil(t, repo, "Failed to create repository")

			dep := &pbinternal.Dependency{
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
					&pbinternal.Dependency{
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proxyServer := httptest.NewServer(tt.mockProxyHandler)
			defer proxyServer.Close()

			sumServer := httptest.NewServer(tt.mockSumHandler)
			defer proxyServer.Close()

			repo := newGoProxySumRepository(proxyServer.URL, sumServer.URL)
			assert.NotNil(t, repo, "Failed to create repository")

			dep := &pbinternal.Dependency{
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

func TestGoModPackageLineHasDependencyAndIndentedString(t *testing.T) {
	t.Parallel()

	dep := &pbinternal.Dependency{
		Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_GO,
		Name:      "github.com/pkg/errors",
		Version:   "v0.9.0",
	}
	tests := []struct {
		name       string
		line       string
		match      bool
		suggestion string
	}{
		{name: "require directive keeps the keyword", line: "require github.com/pkg/errors v0.9.0",
			match: true, suggestion: "require github.com/pkg/errors v0.9.1"},
		{name: "require block keeps the tab", line: "\tgithub.com/pkg/errors v0.9.0",
			match: true, suggestion: "\tgithub.com/pkg/errors v0.9.1"},
		{name: "comment is kept", line: "\tgithub.com/pkg/errors v0.9.0 // pinned for #123",
			match: true, suggestion: "\tgithub.com/pkg/errors v0.9.1 // pinned for #123"},
		{name: "module with a longer path", line: "\tgithub.com/pkg/errors/v2 v0.9.0"},
		{name: "longer version", line: "\tgithub.com/pkg/errors v0.9.01"},
		{name: "replace directive", line: "replace github.com/pkg/errors v0.9.0 => ../errors"},
		{name: "indirect dependency at another version", line: "\tgithub.com/pkg/errors v0.8.0 // indirect"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gmp := &goModPackage{oldVersion: dep.Version, Name: dep.Name, Version: "v0.9.1"}
			require.Equal(t, tt.match, gmp.LineHasDependency(tt.line))
			if tt.match {
				assert.Equal(t, tt.suggestion, gmp.IndentedString(countLeadingWhitespace(tt.line), tt.line, dep))
			}
		})
	}
}
