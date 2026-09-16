// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"
	"github.com/stretchr/testify/require"
)

// TestImageInfoRegoInputShape verifies that []imageInfo survives the exact
// conversion the rego evaluator applies to interfaces.Ingested.Object before
// handing it to a policy as input.ingested. internal/engine/eval/rego/eval.go's
// Eval sets Input.Ingested = res.Object and passes the Input to
// rego.EvalInput; OPA's rego package then round-trips that value through
// encoding/json (util.RoundTrip) before converting it to an ast.Value via
// ast.InterfaceToValue -- so the conversion is JSON-tag-aware, not raw
// reflection. Since imageInfo's own Identity/Verification fields carry no
// json tags, the round trip is expected to preserve "Identity" and
// "Verification" as capitalized top-level keys, matching the shape
// previously produced by the map[string]any this struct replaced.
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
	}

	// Mirror eval.go: Input.Ingested is `any`, populated with the ingester's
	// []imageInfo result and handed to rego.EvalInput.
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
}
