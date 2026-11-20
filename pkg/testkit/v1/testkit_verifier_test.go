// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"context"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/verifier/verifyif"
	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
)

func signedFixture(san string) tkv1.VerificationFixture {
	return tkv1.VerificationFixture{
		IsSigned:   true,
		IsVerified: true,
		Result: &verify.VerificationResult{
			Signature: &verify.SignatureVerificationResult{
				Certificate: &certificate.Summary{SubjectAlternativeName: san},
			},
		},
	}
}

func TestArtifactVerifierNoFixture(t *testing.T) {
	t.Parallel()

	v, ok := tkv1.NewTestKit().ArtifactVerifier()
	assert.False(t, ok, "no artifact configured, so the caller should fall back")
	assert.Nil(t, v)
}

func TestArtifactVerifierByDigest(t *testing.T) {
	t.Parallel()

	const otherManifest = `{"schemaVersion":2,"config":{},"layers":[{}]}`
	tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{
		Versions: []tkv1.ArtifactVersionFixture{
			{
				Tags:         []string{"signed"},
				Manifest:     []byte(indexManifest),
				Verification: []tkv1.VerificationFixture{signedFixture("https://example.test/build")},
			},
			{Tags: []string{"no-verification"}, Manifest: []byte(otherManifest)},
		},
	}))
	v, ok := tk.ArtifactVerifier()
	require.True(t, ok)
	ctx := context.Background()

	// Looked up by the digest computed from the manifest.
	res, err := v.Verify(ctx, verifyif.ArtifactTypeContainer, "stacklok", "minder", digestOf(indexManifest))
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.True(t, res[0].IsSigned)
	assert.True(t, res[0].IsVerified)
	require.NotNil(t, res[0].Signature)
	assert.Equal(t, "https://example.test/build", res[0].Signature.Certificate.SubjectAlternativeName)

	// A version without Verification fixtures is unsigned, like the real
	// verifier's result for an image with no signature.
	res, err = v.Verify(ctx, verifyif.ArtifactTypeContainer, "stacklok", "minder", digestOf(otherManifest))
	require.NoError(t, err)
	assert.Equal(t, []verifyif.Result{{}}, res)

	// No version with this digest: no results.
	res, err = v.Verify(ctx, verifyif.ArtifactTypeContainer, "stacklok", "minder", "sha256:missing")
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Empty(t, res)

	// Results are copies; changing them does not change later lookups.
	res, err = v.VerifyContainer(ctx, "stacklok", "minder", digestOf(indexManifest))
	require.NoError(t, err)
	res[0].IsVerified = false
	res, err = v.VerifyContainer(ctx, "stacklok", "minder", digestOf(indexManifest))
	require.NoError(t, err)
	assert.True(t, res[0].IsVerified)
}

func TestArtifactVerifierRejectsInvalidFixture(t *testing.T) {
	t.Parallel()

	tests := map[string]tkv1.VerificationFixture{
		"verified without signature": {IsSigned: true, IsVerified: true, Result: &verify.VerificationResult{}},
		"verified without result":    {IsSigned: true, IsVerified: true},
		"verified but unsigned":      {IsVerified: true},
	}
	for name, fixture := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{
				Versions: []tkv1.ArtifactVersionFixture{{
					Manifest:     []byte(indexManifest),
					Verification: []tkv1.VerificationFixture{fixture},
				}},
			}))
			v, ok := tk.ArtifactVerifier()
			require.True(t, ok)
			_, err := v.Verify(context.Background(), verifyif.ArtifactTypeContainer,
				"stacklok", "minder", digestOf(indexManifest))
			require.Error(t, err)
		})
	}
}

func TestArtifactVerifierUnsupportedType(t *testing.T) {
	t.Parallel()

	tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{}))
	v, ok := tk.ArtifactVerifier()
	require.True(t, ok)
	_, err := v.Verify(context.Background(), verifyif.ArtifactType("npm"), "o", "n", "sha256:1")
	require.Error(t, err)
}
