// Copyright 2023 Stacklok, Inc
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

// Package verifyif provides the interface for artifact verifiers, including
// the Result type
package verifyif

import (
	"context"

	"github.com/sigstore/sigstore-go/pkg/verify"
)

// Result is the result of the verification
type Result struct {
	IsSigned         bool   `json:"is_signed"`
	IsVerified       bool   `json:"is_verified"`
	IsBundleVerified bool   `json:"is_bundle_verified"`
	URI              string `json:"uri"`
	verify.VerificationResult
}

// GetUri returns the URI of the artifact
// explicit getter because the URI is something we might want to try
// displaying even if the result is nil as part of error handling
func (r *Result) GetUri() string {
	if r == nil {
		return ""
	}
	return r.URI
}

// ArtifactVerifier is the interface for artifact verifiers
type ArtifactVerifier interface {
	VerifyContainer(ctx context.Context,
		registry, owner, artifact, version string) (*Result, error)
}
