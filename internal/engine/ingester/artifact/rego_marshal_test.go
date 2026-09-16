// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"
	"github.com/stretchr/testify/require"

	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// TestImageInfoRegoInputShape checks []imageInfo round-trips through OPA's
// actual conversion path (eval.go -> rego.EvalInput -> util.RoundTrip ->
// ast.InterfaceToValue) and still exposes Identity/Verification/Manifest
// as top-level keys.
func TestImageInfoRegoInputShape(t *testing.T) {
	t.Parallel()

	info := imageInfo{
		Identity: imageRef{
			Registry:   "ghcr.io",
			Repository: "stacklok/test",
			Tags:       []string{"latest"},
			Digest:     "sha256:1234",
		},
		Verification: verification{
			IsSigned:   true,
			IsVerified: true,
			Repository: "https://github.com/stacklok/test",
		},
		Manifest: &provifv1.RawManifest{
			Descriptor: v1.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    v1.Hash{Algorithm: "sha256", Hex: "1234"},
			},
			Content: []byte(`{"schemaVersion":2}`),
		},
	}

	raw := util.Reference(any([]imageInfo{info}))
	require.NoError(t, util.RoundTrip(raw))

	val, err := ast.InterfaceToValue(*raw)
	require.NoError(t, err)

	out, err := ast.JSON(val)
	require.NoError(t, err)

	results, ok := out.([]any)
	require.True(t, ok, "expected top-level array")
	require.Len(t, results, 1)

	entry, ok := results[0].(map[string]any)
	require.True(t, ok, "expected each entry to be a JSON object")

	identity, ok := entry["Identity"].(map[string]any)
	require.True(t, ok, "expected Identity key to survive the rego round trip")
	require.Equal(t, "ghcr.io", identity["registry"])
	require.Equal(t, "stacklok/test", identity["repository"])
	require.Equal(t, "sha256:1234", identity["digest"])

	verificationOut, ok := entry["Verification"].(map[string]any)
	require.True(t, ok, "expected Verification key to survive the rego round trip")
	require.Equal(t, true, verificationOut["is_signed"])
	require.Equal(t, true, verificationOut["is_verified"])

	manifestOut, ok := entry["Manifest"].(map[string]any)
	require.True(t, ok, "expected Manifest key to survive the rego round trip")
	require.Equal(t, "application/vnd.oci.image.manifest.v1+json", manifestOut["mediaType"])
	require.Equal(t, "sha256:1234", manifestOut["digest"])
	require.NotEmpty(t, manifestOut["content"])
}