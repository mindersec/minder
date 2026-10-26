// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"fmt"

	"github.com/mindersec/minder/internal/verifier/verifyif"
)

// Ensure that TestKit supplies its own artifact verifier, so the artifact
// ingester never builds a real sigstore verifier against it.
var _ verifyif.ArtifactVerifierProvider = &TestKit{}

// ArtifactVerifier implements verifyif.ArtifactVerifierProvider. It returns
// a verifier backed by the Verification fixtures of the artifact set with
// WithArtifact, or false if no artifact is configured.
func (tk *TestKit) ArtifactVerifier() (verifyif.ArtifactVerifier, bool) {
	if tk.artifact == nil {
		return nil, false
	}
	return &fixtureVerifier{tk: tk}, true
}

// fixtureVerifier serves canned verification results by digest.
type fixtureVerifier struct {
	tk *TestKit
}

// Verify returns the results for the version whose digest is checksumref.
// A version with no Verification fixtures is reported as unsigned, as the
// real verifier does. A digest that matches no version returns no results.
func (f *fixtureVerifier) Verify(
	ctx context.Context, artifactType verifyif.ArtifactType, owner, name, checksumref string,
) ([]verifyif.Result, error) {
	if artifactType != verifyif.ArtifactTypeContainer {
		return nil, fmt.Errorf("unsupported artifact type %q in TestKit", artifactType)
	}
	return f.VerifyContainer(ctx, owner, name, checksumref)
}

// VerifyContainer implements verifyif.ArtifactVerifier.
func (f *fixtureVerifier) VerifyContainer(
	_ context.Context, _, _, checksumref string,
) ([]verifyif.Result, error) {
	versions, err := f.tk.resolveArtifactVersions()
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.digest == checksumref {
			return cloneResults(v.verification), nil
		}
	}
	return []verifyif.Result{}, nil
}

// verificationResults maps fixtures to verifier results, rejecting any
// combination a real verifier could not return. No fixtures means unsigned.
func verificationResults(fixtures []VerificationFixture) ([]verifyif.Result, error) {
	if len(fixtures) == 0 {
		return []verifyif.Result{{}}, nil
	}
	out := make([]verifyif.Result, 0, len(fixtures))
	for i, f := range fixtures {
		if err := f.Validate(); err != nil {
			return nil, fmt.Errorf("verification %d: %w", i, err)
		}
		res := verifyif.Result{IsSigned: f.IsSigned, IsVerified: f.IsVerified}
		if f.Result != nil {
			res.VerificationResult = *f.Result
		}
		out = append(out, res)
	}
	return out, nil
}

func cloneResults(in []verifyif.Result) []verifyif.Result {
	out := make([]verifyif.Result, len(in))
	copy(out, in)
	return out
}
