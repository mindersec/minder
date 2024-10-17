// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package verifyif provides the interface for artifact verifiers, including
// the Result type
package verifyif

import (
	"context"

	"github.com/sigstore/sigstore-go/pkg/verify"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// ArtifactType represents the type of artifact, i.e., container, npm, etc.
type ArtifactType string

const (
	// ArtifactTypeContainer is a container artifact
	ArtifactTypeContainer ArtifactType = "container"
)

// Result is the result of the verification
type Result struct {
	IsSigned   bool `json:"is_signed"`
	IsVerified bool `json:"is_verified"`
	verify.VerificationResult
}

// ArtifactVerifier is the interface for artifact verifiers
type ArtifactVerifier interface {
	Verify(ctx context.Context, artifactType ArtifactType,
		owner, name, checksumref string) ([]Result, error)
	VerifyContainer(ctx context.Context,
		owner, artifact, checksumref string) ([]Result, error)
}
