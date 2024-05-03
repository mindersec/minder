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
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/credentials"
	mock_ghclient "github.com/stacklok/minder/internal/providers/github/mock"
	"github.com/stacklok/minder/internal/verifier/verifyif"
	mockverify "github.com/stacklok/minder/internal/verifier/verifyif/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func testGithubProviderBuilder() *providers.ProviderBuilder {
	const (
		ghApiUrl = "https://api.github.com"
	)

	baseURL := ghApiUrl + "/"

	definitionJSON := `{
		"github": {
			"endpoint": "` + baseURL + `"
		}
	}`

	return providers.NewProviderBuilder(
		&db.Provider{
			Name:       "github",
			Version:    provifv1.V1,
			Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeRest, db.ProviderTypeGit},
			Definition: json.RawMessage(definitionJSON),
		},
		sql.NullString{},
		false,
		credentials.NewGitHubTokenCredential("token"),
		&serverconfig.ProviderConfig{},
		nil, // this is unused here
	)
}

func TestArtifactIngestMatching(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	tests := []struct {
		name          string
		wantErr       bool
		wantNonNilRes bool
		errType       error
		mockSetup     func(*mock_ghclient.MockGitHub, *mockverify.MockArtifactVerifier)
		artifact      *pb.Artifact
		params        map[string]interface{}
	}{
		{
			name:          "matching-name",
			wantErr:       false,
			wantNonNilRes: true,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"latest"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
					}, nil)
				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, verifyif.ArtifactRegistry(""), "stacklok", "matching-name", "sha256:1234").
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
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name-and-tag").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"latest"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"main", "production"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:5678"),
						},
					}, nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, verifyif.ArtifactRegistry(""), "stacklok", "matching-name-and-tag", "sha256:1234").
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
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name-but-not-tags").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"main", "production"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"dev"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:5678"),
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
				"tags": []string{"latest"},
			},
		},
		{
			name:          "multiple-tags-from-different-versions",
			wantErr:       true,
			wantNonNilRes: false,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name-but-not-tags").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"main", "production"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"dev"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:5678"),
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
				"tags": []string{"latest", "dev"},
			},
		},
		{
			name:          "multiple-tags-from-same-version",
			wantErr:       true,
			wantNonNilRes: false,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name-but-not-tags").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"main", "production"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"dev"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:5678"),
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
			name:          "matching-multiple-tags-from-same-version",
			wantErr:       false,
			wantNonNilRes: true,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name-but-not-tags").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"main", "production", "dev"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"v1.0.0"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:5678"),
						},
					}, nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, verifyif.ArtifactRegistry(""), "stacklok", "matching-name-but-not-tags", "sha256:1234").
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
			mockSetup: func(_ *mock_ghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
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
		{
			name:          "match-any-name",
			wantErr:       false,
			wantNonNilRes: true,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"latest"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
					}, nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, verifyif.ArtifactRegistry(""), "stacklok", "matching-name", "sha256:1234").
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
				"name": "", // empty string means match any name
			},
		},
		{
			name:          "test-matching-regex",
			wantErr:       false,
			wantNonNilRes: true,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"v1.0.0"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
					}, nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, verifyif.ArtifactRegistry(""), "stacklok", "matching-name", "sha256:1234").
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
				"name":      "matching-name",
				"tag_regex": "^v[0-9]+\\.[0-9]+\\.[0-9]+$",
			},
		},
		{
			name:          "test-matching-regex-with-multiple-tags",
			wantErr:       false,
			wantNonNilRes: true,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, mockVerifier *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"v2.0.0", "latest"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
					}, nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), verifyif.ArtifactTypeContainer, verifyif.ArtifactRegistry(""), "stacklok", "matching-name", "sha256:1234").
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
				"name":      "matching-name",
				"tag_regex": "^v[0-9]+\\.[0-9]+\\.[0-9]+$",
			},
		},
		{
			name:          "tag-doesnt-match-regex",
			wantErr:       true,
			wantNonNilRes: false,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name-but-not-tags").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{"latest"},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
					}, nil)
			},
			artifact: &pb.Artifact{
				Type:  "container",
				Name:  "matching-name-but-not-tags",
				Owner: "stacklok",
			},
			params: map[string]interface{}{
				"name":      "matching-name-but-not-tags",
				"tag_regex": "^v[0-9]+\\.[0-9]+\\.[0-9]+$",
			},
		},
		{
			name:          "multiple-tags-doesnt-match-regex",
			wantErr:       true,
			wantNonNilRes: false,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name-but-not-tags").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{
										"latest",
										"pr-123",
										"testing",
									},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
						},
					}, nil)
			},
			artifact: &pb.Artifact{
				Type:  "container",
				Name:  "matching-name-but-not-tags",
				Owner: "stacklok",
			},
			params: map[string]interface{}{
				"name":      "matching-name-but-not-tags",
				"tag_regex": "^v[0-9]+\\.[0-9]+\\.[0-9]+$",
			},
		},
		{
			name:          "test-artifact-with-empty-tags",
			wantErr:       true,
			wantNonNilRes: false,
			mockSetup: func(_ *mock_ghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
			},
			artifact: &pb.Artifact{
				Type:  "container",
				Name:  "matching-name-but-not-tags",
				Owner: "stacklok",
			},
			params: map[string]interface{}{
				"name": "matching-name-but-not-tags",
				"tags": []string{""},
			},
		},
		{
			name:          "test-artifact-version-with-no-tags",
			wantErr:       true,
			wantNonNilRes: false,
			mockSetup: func(mockGhClient *mock_ghclient.MockGitHub, _ *mockverify.MockArtifactVerifier) {
				mockGhClient.EXPECT().
					GetPackageVersions(gomock.Any(), "stacklok", "container", "matching-name-but-not-tags").
					Return([]*github.PackageVersion{
						{
							Metadata: &github.PackageMetadata{
								Container: &github.PackageContainerMetadata{
									Tags: []string{},
								},
							},
							CreatedAt: &github.Timestamp{Time: time.Now()},
							Name:      github.String("sha256:1234"),
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
				"tags": []string{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockGhClient := mock_ghclient.NewMockGitHub(ctrl)
			mockVerifier := mockverify.NewMockArtifactVerifier(ctrl)

			ing, err := NewArtifactDataIngest(nil, testGithubProviderBuilder())
			require.NoError(t, err, "expected no error")

			ing.ghCli = mockGhClient
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
			"build-signer-uri",
			&certificate.Summary{
				Extensions: certificate.Extensions{
					BuildSignerURI: "https://github.com/openvex/vexctl/.github/workflows/release.yaml@refs/tags/v0.2.6",
				},
			},
			"/openvex/vexctl/.github/workflows/release.yaml",
			false,
		},
		{
			"san-uri",
			&certificate.Summary{
				SubjectAlternativeName: certificate.SubjectAlternativeName{
					Type:  certificate.SubjectAlternativeNameTypeURI,
					Value: "https://github.com/openvex/vexctl/.github/workflows/release.yaml@refs/tags/v0.2.6",
				},
			},
			"/openvex/vexctl/.github/workflows/release.yaml",
			false,
		},
		{
			"san-email",
			&certificate.Summary{
				SubjectAlternativeName: certificate.SubjectAlternativeName{
					Type:  certificate.SubjectAlternativeNameTypeEmail,
					Value: "test@example.com",
				},
			},
			"test@example.com",
			false,
		},
		{
			"san-other",
			&certificate.Summary{
				SubjectAlternativeName: certificate.SubjectAlternativeName{
					Type:  certificate.SubjectAlternativeNameTypeEmail,
					Value: "Hello Friend",
				},
			},
			"Hello Friend",
			false,
		},
		{
			"error-invalid-url",
			&certificate.Summary{
				SubjectAlternativeName: certificate.SubjectAlternativeName{
					Type:  certificate.SubjectAlternativeNameTypeURI,
					Value: "http:\n\\/",
				},
			},
			"",
			true,
		},
		{
			"no-values",
			&certificate.Summary{},
			"",
			false,
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
