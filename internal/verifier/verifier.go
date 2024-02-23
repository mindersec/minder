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

// Package verifier provides a client for verifying various types of artifacts against various provenance mechanisms
package verifier

import (
	"fmt"
	"strings"

	"github.com/stacklok/minder/internal/verifier/sigstore"
	"github.com/stacklok/minder/internal/verifier/sigstore/container"
	"github.com/stacklok/minder/internal/verifier/verifyif"
)

const (
	// ArtifactSignatureSuffix is the suffix for the signature tag
	ArtifactSignatureSuffix = ".sig"
)

// Type represents the type of verifier, i.e., sigstore, slsa, etc.
type Type string

const (
	// VerifierSigstore is the sigstore verifier
	VerifierSigstore Type = "sigstore"
)

// NewVerifier creates a new Verifier object
func NewVerifier(verifier Type, verifierURL string, containerAuth ...container.AuthMethod) (verifyif.ArtifactVerifier, error) {
	var err error
	var v verifyif.ArtifactVerifier

	// create the verifier
	switch verifier {
	case VerifierSigstore:
		v, err = sigstore.New(verifierURL, containerAuth...)
		if err != nil {
			return nil, fmt.Errorf("error creating sigstore verifier: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown verifier type: %s", verifier)
	}

	// return the verifier
	return v, nil
}

// GetSignatureTag returns the signature tag for a given image, if exists, otherwise empty string
func GetSignatureTag(tags []string) string {
	// if the artifact has a .sig tag it's a signature, skip it
	for _, tag := range tags {
		if strings.HasSuffix(tag, ArtifactSignatureSuffix) {
			return tag
		}
	}
	return ""
}
