// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"

	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
)

const testIndex = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[]}`

func sha256Digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// evalArtifactExpr evaluates a Starlark expression with the artifact
// builtins predeclared and returns the result.
func evalArtifactExpr(t *testing.T, expr string) (starlark.Value, error) {
	t.Helper()

	predeclared := starlark.StringDict{
		"artifact":          starlark.NewBuiltin("artifact", builtinArtifact),
		"version":           starlark.NewBuiltin("version", builtinVersion),
		"signed":            starlark.NewBuiltin("signed", builtinSigned),
		"unsigned":          starlark.NewBuiltin("unsigned", builtinUnsigned),
		"verification_json": starlark.NewBuiltin("verification_json", builtinVerificationJSON),
		"INDEX":             starlark.String(testIndex),
	}
	thread := &starlark.Thread{Name: t.Name()}
	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, "test.star", "res = "+expr, predeclared)
	if err != nil {
		return nil, err
	}
	return globals["res"], nil
}

func evalArtifact(t *testing.T, expr string) tkv1.ArtifactFixture {
	t.Helper()

	val, err := evalArtifactExpr(t, expr)
	require.NoError(t, err)
	art, ok := val.(*ArtifactValue)
	require.True(t, ok, "expected artifact, got %s", val.Type())
	return art.Fixture
}

func TestArtifactBuiltin(t *testing.T) {
	t.Parallel()

	fixture := evalArtifact(t, `artifact(
		registry="ghcr.io",
		versions=[
			version(tags=["v1.0.0", "latest"], manifest=INDEX, created_at="2026-01-02T03:04:05Z"),
			version(tags=["v0.9.0"], digest="sha256:abcd"),
		],
	)`)

	assert.Equal(t, "ghcr.io", fixture.Registry)
	require.Len(t, fixture.Versions, 2)

	v1 := fixture.Versions[0]
	assert.Equal(t, []string{"v1.0.0", "latest"}, v1.Tags)
	assert.Equal(t, []byte(testIndex), v1.Manifest)
	assert.Equal(t, sha256Digest(testIndex), v1.Digest, "digest is computed from the manifest")
	assert.Equal(t, "application/vnd.oci.image.index.v1+json", v1.MediaType)
	assert.True(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Equal(v1.CreatedAt))
	assert.Empty(t, v1.Verification)

	v2 := fixture.Versions[1]
	assert.Equal(t, []string{"v0.9.0"}, v2.Tags)
	assert.Nil(t, v2.Manifest)
	assert.Equal(t, "sha256:abcd", v2.Digest)
	assert.True(t, v2.CreatedAt.IsZero(), "created_at is left unset so TestKit defaults it to now")
}

func TestArtifactBuiltinEmpty(t *testing.T) {
	t.Parallel()

	fixture := evalArtifact(t, `artifact()`)
	assert.Equal(t, tkv1.ArtifactFixture{}, fixture)
}

func TestVersionBuiltinMatchingDigest(t *testing.T) {
	t.Parallel()

	fixture := evalArtifact(t,
		`artifact(versions=[version(tags=["v1"], manifest=INDEX, digest="`+sha256Digest(testIndex)+`")])`)
	assert.Equal(t, sha256Digest(testIndex), fixture.Versions[0].Digest)
}

func TestVerificationBuiltins(t *testing.T) {
	t.Parallel()

	fixture := evalArtifact(t, `artifact(versions=[
		version(tags=["unsigned"], digest="sha256:1", verification=unsigned()),
		version(tags=["signed"], digest="sha256:2", verification=signed(
			san="https://github.com/stacklok/minder/.github/workflows/build.yml@refs/heads/main",
			issuer="https://token.actions.githubusercontent.com",
			source_repository_uri="https://github.com/stacklok/minder",
			source_repository_ref="refs/heads/main",
			runner_environment="github-hosted",
		)),
		version(tags=["unverified"], digest="sha256:3", verification=signed(verified=False)),
		version(tags=["attested"], digest="sha256:4", verification=[
			signed(issuer="https://token.actions.githubusercontent.com"),
			signed(predicate_type="https://slsa.dev/provenance/v1",
				predicate={"buildDefinition": {"buildType": "https://actions.github.io/buildtypes/workflow/v1"}}),
		]),
	])`)
	require.Len(t, fixture.Versions, 4)

	// unsigned()
	require.Len(t, fixture.Versions[0].Verification, 1)
	assert.Equal(t, tkv1.VerificationFixture{}, fixture.Versions[0].Verification[0])

	// signed() with certificate fields
	require.Len(t, fixture.Versions[1].Verification, 1)
	ver := fixture.Versions[1].Verification[0]
	assert.True(t, ver.IsSigned)
	assert.True(t, ver.IsVerified)
	require.NotNil(t, ver.Result)
	require.NotNil(t, ver.Result.Signature)
	cert := ver.Result.Signature.Certificate
	require.NotNil(t, cert)
	assert.Equal(t, "https://github.com/stacklok/minder/.github/workflows/build.yml@refs/heads/main",
		cert.SubjectAlternativeName)
	assert.Equal(t, "https://token.actions.githubusercontent.com", cert.Issuer)
	assert.Equal(t, "https://github.com/stacklok/minder", cert.SourceRepositoryURI)
	assert.Equal(t, "refs/heads/main", cert.SourceRepositoryRef)
	assert.Equal(t, "github-hosted", cert.RunnerEnvironment)
	assert.Nil(t, ver.Result.Statement)

	// signed(verified=False)
	require.Len(t, fixture.Versions[2].Verification, 1)
	assert.Equal(t, tkv1.VerificationFixture{IsSigned: true}, fixture.Versions[2].Verification[0])

	// a list of results, one with an attestation
	require.Len(t, fixture.Versions[3].Verification, 2)
	att := fixture.Versions[3].Verification[1]
	require.NotNil(t, att.Result)
	require.NotNil(t, att.Result.Statement)
	// Verified by default, so it carries an (empty) certificate.
	require.NotNil(t, att.Result.Signature)
	assert.Equal(t, certificate.Summary{}, *att.Result.Signature.Certificate)
	assert.Equal(t, "https://slsa.dev/provenance/v1", att.Result.Statement.GetPredicateType())
	assert.Equal(t, "https://actions.github.io/buildtypes/workflow/v1",
		att.Result.Statement.GetPredicate().GetFields()["buildDefinition"].
			GetStructValue().GetFields()["buildType"].GetStringValue())
}

func TestSignedAlwaysProducesValidFixture(t *testing.T) {
	t.Parallel()

	// signed() with no certificate fields used to leave Signature nil on a
	// verified result, which crashed the artifact ingester.
	fixture := evalArtifact(t, `artifact(versions=[
		version(tags=["bare"], digest="sha256:1", verification=signed()),
		version(tags=["unverified"], digest="sha256:2", verification=signed(verified=False)),
		version(tags=["unsigned"], digest="sha256:3", verification=unsigned()),
	])`)
	for _, v := range fixture.Versions {
		for _, ver := range v.Verification {
			assert.NoError(t, ver.Validate(), "tags %v", v.Tags)
		}
	}
}

const verificationResultJSON = `{
  "mediaType": "application/vnd.dev.sigstore.verificationresult+json;version=0.1",
  "statement": {
    "_type": "https://in-toto.io/Statement/v1",
    "subject": [{"name": "ghcr.io/stacklok/minder", "digest": {"sha256": "abcd"}}],
    "predicateType": "https://slsa.dev/provenance/v1",
    "predicate": {"buildDefinition": {"buildType": "https://actions.github.io/buildtypes/workflow/v1"}}
  },
  "signature": {
    "certificate": {
      "subjectAlternativeName": "https://github.com/stacklok/minder/.github/workflows/build.yml@refs/heads/main",
      "issuer": "https://token.actions.githubusercontent.com",
      "sourceRepositoryURI": "https://github.com/stacklok/minder",
      "sourceRepositoryRef": "refs/heads/main",
      "runnerEnvironment": "github-hosted"
    }
  }
}`

const signatureOnlyJSON = `{
  "mediaType": "application/vnd.dev.sigstore.verificationresult+json;version=0.1",
  "signature": {"certificate": {"subjectAlternativeName": "user@example.com", "issuer": "https://accounts.google.com"}}
}`

func TestVerificationJSONBuiltin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantSAN   string
		wantStmt  bool
	}{
		{
			name:      "single result",
			content:   verificationResultJSON,
			wantCount: 1,
			wantSAN:   "https://github.com/stacklok/minder/.github/workflows/build.yml@refs/heads/main",
			wantStmt:  true,
		},
		{
			name: "list of wrapped results",
			content: `[{"verificationResult": ` + verificationResultJSON + `}, ` +
				`{"verificationResult": ` + verificationResultJSON + `}]`,
			wantCount: 2,
			wantSAN:   "https://github.com/stacklok/minder/.github/workflows/build.yml@refs/heads/main",
			wantStmt:  true,
		},
		{
			name:      "signature without statement",
			content:   signatureOnlyJSON,
			wantCount: 1,
			wantSAN:   "user@example.com",
			wantStmt:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			val, err := builtinVerificationJSON(nil, starlark.NewBuiltin("verification_json", nil),
				starlark.Tuple{starlark.String(tt.content)}, nil)
			require.NoError(t, err)
			ver, ok := val.(*VerificationValue)
			require.True(t, ok)
			require.Len(t, ver.Fixtures, tt.wantCount)

			for _, f := range ver.Fixtures {
				assert.True(t, f.IsSigned)
				assert.True(t, f.IsVerified)
				require.NotNil(t, f.Result)
				require.NotNil(t, f.Result.Signature)
				assert.Equal(t, tt.wantSAN, f.Result.Signature.Certificate.SubjectAlternativeName)
				if tt.wantStmt {
					require.NotNil(t, f.Result.Statement)
					assert.Equal(t, "https://slsa.dev/provenance/v1", f.Result.Statement.GetPredicateType())
				} else {
					assert.Nil(t, f.Result.Statement)
				}
			}
		})
	}
}

func TestArtifactBuiltinErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		expr   string
		errMsg string
	}{
		{
			name:   "digest mismatch",
			expr:   `version(tags=["v1"], manifest=INDEX, digest="sha256:wrong")`,
			errMsg: "does not match manifest content digest",
		},
		{
			name:   "missing manifest and digest",
			expr:   `version(tags=["v1"])`,
			errMsg: "either a manifest or a digest is required",
		},
		{
			name:   "manifest is not JSON",
			expr:   `version(tags=["v1"], manifest="not json")`,
			errMsg: "parsing manifest",
		},
		{
			name:   "tags are required",
			expr:   `version(digest="sha256:1")`,
			errMsg: "missing argument for tags",
		},
		{
			name:   "non-string tag",
			expr:   `version(tags=[1], digest="sha256:1")`,
			errMsg: "tags must be a list of strings",
		},
		{
			name:   "invalid created_at",
			expr:   `version(tags=["v1"], digest="sha256:1", created_at="yesterday")`,
			errMsg: "invalid created_at",
		},
		{
			name:   "invalid verification",
			expr:   `version(tags=["v1"], digest="sha256:1", verification="signed")`,
			errMsg: "verification must be a verification",
		},
		{
			name:   "invalid verification list entry",
			expr:   `version(tags=["v1"], digest="sha256:1", verification=[unsigned(), "x"])`,
			errMsg: "verification[1] must be a verification",
		},
		{
			name:   "certificate fields on unverified signature",
			expr:   `signed(verified=False, san="user@example.com")`,
			errMsg: "certificate fields require verified=True",
		},
		{
			name:   "versions must be version values",
			expr:   `artifact(versions=["v1"])`,
			errMsg: "versions[0] must be a version",
		},
		{
			name:   "verification_json is not JSON",
			expr:   `verification_json("not json")`,
			errMsg: "verification_json: result 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := evalArtifactExpr(t, tt.expr)
			require.ErrorContains(t, err, tt.errMsg)
		})
	}
}

func TestEvalRejectsNonArtifact(t *testing.T) {
	t.Parallel()

	tr := NewRunner().newTestCaseRunner("test", fstest.MapFS{}, ".", nil)
	_, err := tr.runFile("test.star", `eval(rule="r", artifacts="not an artifact")`)
	require.ErrorContains(t, err, "artifacts")
}
