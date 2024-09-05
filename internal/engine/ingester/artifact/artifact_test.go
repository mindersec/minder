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

package artifact

import (
	"context"
	"testing"
	"time"

	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/github/clients"
	mockghclient "github.com/stacklok/minder/internal/providers/github/mock"
	"github.com/stacklok/minder/internal/providers/github/properties"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/verifier/verifyif"
	mockverify "github.com/stacklok/minder/internal/verifier/verifyif/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func testGithubProvider() (provinfv1.GitHub, error) {
	const (
		ghApiUrl = "https://api.github.com"
	)

	baseURL := ghApiUrl + "/"

	return clients.NewRestClient(
		&pb.GitHubProviderConfig{
			Endpoint: &baseURL,
		},
		nil,
		nil,
		&ratecache.NoopRestClientCache{},
		credentials.NewGitHubTokenCredential("token"),
		clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
		properties.NewPropertyFetcherFactory(),
		"",
	)
}

func TestArtifactIngestMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		wantErr       bool
		wantNonNilRes bool
		errType       error
		mockSetup     func(*mockghclient.MockGitHub, *mockverify.MockArtifactVerifier)
		artifact      *pb.Artifact
		params        map[string]interface{}
	}{
		{
			name:          "matching-name",
			wantErr:       false,
			wantNonNilRes: true,
			mockSetup: func(mockGhClient *mockghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetArtifactVersions(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*pb.ArtifactVersion{
						{
							Sha:       "sha256:1234",
							Tags:      []string{"latest"},
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)
				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, "stacklok", "matching-name", "sha256:1234").
					Return([]verifyif.Result{
						{
							IsSigned:   false,
							IsVerified: false,
						},
					}, nil)
			},
			artifact: &pb.Artifact{
				Type:  "container",
				Name:  "matching-name",
				Owner: "stacklok",
			},
			params: map[string]interface{}{
				"name": "matching-name",
				// missing tags means wildcard match any tag
			},
		},
		{
			name:          "matching-name-and-tag",
			wantErr:       false,
			wantNonNilRes: true,
			mockSetup: func(mockGhClient *mockghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetArtifactVersions(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*pb.ArtifactVersion{
						{
							Sha:       "sha256:1234",
							Tags:      []string{"latest"},
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, "stacklok", "matching-name-and-tag", "sha256:1234").
					Return([]verifyif.Result{
						{
							IsSigned:   false,
							IsVerified: false,
						},
					}, nil)
			},
			artifact: &pb.Artifact{
				Type:  "container",
				Name:  "matching-name-and-tag",
				Owner: "stacklok",
			},
			params: map[string]interface{}{
				"name": "matching-name-and-tag",
				"tags": []string{"latest"},
			},
		},
		{
			name:          "matching-name-but-not-tags",
			wantErr:       true,
			wantNonNilRes: false,
			mockSetup: func(mockGhClient *mockghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetArtifactVersions(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*pb.ArtifactVersion{}, nil)
			},
			artifact: &pb.Artifact{
				Type:  "container",
				Name:  "matching-name-but-not-tags",
				Owner: "stacklok",
			},
			params: map[string]interface{}{
				"name": "matching-name-but-not-tags",
				"tags": []string{"latest"},
			},
		},
		// test "multiple-tags-from-different-versions" was removed since
		// filtering is no longer tested here, but instead in the versionsfilter_test.go
		// Test "multiple-tags-from-same-version" was removed since
		// filtering is no longer tested here, but instead in the versionsfilter_test.go
		{
			name:          "matching-multiple-tags-from-same-version",
			wantErr:       false,
			wantNonNilRes: true,
			mockSetup: func(mockGhClient *mockghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetArtifactVersions(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*pb.ArtifactVersion{
						{
							Sha:       "sha256:1234",
							Tags:      []string{"main", "production", "dev"},
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, "stacklok", "matching-name-but-not-tags", "sha256:1234").
					Return([]verifyif.Result{
						{
							IsSigned:   false,
							IsVerified: false,
						},
					}, nil)
			},
			artifact: &pb.Artifact{
				Type:  "container",
				Name:  "matching-name-but-not-tags",
				Owner: "stacklok",
			},
			params: map[string]interface{}{
				"name": "matching-name-but-not-tags",
				"tags": []string{"main", "production", "dev"},
			},
		},
		{
			name:          "not-matching-name",
			wantErr:       true,
			wantNonNilRes: false,
			errType:       evalerrors.ErrEvaluationSkipSilently,
			mockSetup: func(_ *mockghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
			},
			artifact: &pb.Artifact{
				Type:  "container",
				Name:  "not-matching-name",
				Owner: "stacklok",
			},
			params: map[string]interface{}{
				"name": "name-does-NOT-match",
			},
		},
		// Test "match-any-name" was removed since filtering is no longer tested here, but instead in the versionsfilter_test.go
		// Test "test-matching-regex" was removed since filtering is no longer tested here, but instead in the versionsfilter_test.go
		// Test "tag-doesnt-match-regex" was removed since filtering is no longer tested here, but instead in the versionsfilter_test.go
		// Test "multiple-tags-doesnt-match-regex" was removed since filtering is no longer tested here, but instead in the versionsfilter_test.go
		// Test "test-artifact-with-empty-tags" was removed since filtering is no longer tested here, but instead in the versionsfilter_test.go
		// Test "test-artifact-version-with-no-tags" was removed since filtering is no longer tested here, but instead in the versionsfilter_test.go
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() {
				ctrl.Finish()
			})

			mockGhClient := mockghclient.NewMockGitHub(ctrl)
			mockVerifier := mockverify.NewMockArtifactVerifier(ctrl)

			prov, err := testGithubProvider()
			require.NoError(t, err)
			ing, err := NewArtifactDataIngest(prov)
			require.NoError(t, err, "expected no error")

			ing.prov = mockGhClient
			ing.artifactVerifier = mockVerifier

			tt.mockSetup(mockGhClient, mockVerifier)

			got, err := ing.Ingest(context.Background(), tt.artifact, tt.params)

			if tt.wantErr {
				require.Error(t, err, "expected error")
			} else {
				require.NoError(t, err, "expected no error")
			}

			if tt.errType != nil {
				require.ErrorIs(t, err, tt.errType, "expected error type")
			}

			if tt.wantNonNilRes {
				require.NotNil(t, got, "expected non-nil result")
			} else {
				require.Nil(t, got, "expected nil result")
			}
		})
	}
}

func TestSignerIdentityFromCertificate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		sut      *certificate.Summary
		expected string
		mustErr  bool
	}{
		{
			"san-uri",
			&certificate.Summary{
				SubjectAlternativeName: "https://github.com/openvex/vexctl/.github/workflows/release.yaml@refs/tags/v0.2.6",
				Extensions: certificate.Extensions{
					Issuer:              githubTokenIssuer,
					SourceRepositoryURI: "https://github.com/openvex/vexctl",
				},
			},
			"/.github/workflows/release.yaml",
			false,
		},
		{
			"build-signer-uri-ignore",
			&certificate.Summary{
				SubjectAlternativeName: "https://github.com/openvex/vexctl/.github/workflows/release.yaml@refs/tags/v0.2.6",
				Extensions: certificate.Extensions{
					Issuer:              githubTokenIssuer,
					BuildSignerURI:      "https://github.com/openvex/vexctl/.github/workflows/fake.yaml@refs/tags/v0.2.6",
					SourceRepositoryURI: "https://github.com/openvex/vexctl",
				},
			},
			"/.github/workflows/release.yaml",
			false,
		},
		{
			"no-source-repo",
			&certificate.Summary{
				SubjectAlternativeName: "https://github.com/openvex/vexctl/.github/workflows/release.yaml@refs/tags/v0.2.9",
				Extensions: certificate.Extensions{
					Issuer: githubTokenIssuer,
				},
			},
			"",
			true,
		},
		{
			"not-from-github-actions", // If URLs were note autenticated from actions, don't parse
			&certificate.Summary{
				SubjectAlternativeName: "https://github.com/openvex/vexctl/.github/workflows/release.yaml@refs/tags/v0.2.7",
				Extensions: certificate.Extensions{
					SourceRepositoryURI: "https://github.com/openvex/vexctl",
				},
			},
			"https://github.com/openvex/vexctl/.github/workflows/release.yaml@refs/tags/v0.2.7",
			false,
		},
		{
			"no-values",
			&certificate.Summary{},
			"",
			true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			identity, err := signerIdentityFromCertificate(tc.sut)
			if tc.mustErr {
				require.Error(t, err, identity)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, identity)
		})
	}
}
