// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mindersec/minder/internal/verifier/verifyif"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// DefaultTag is the tag used when an image reference does not specify one.
const DefaultTag = "latest"

// ParseImageRef splits a namespace-relative "repo[:tag]" reference, defaulting the tag to DefaultTag.
func ParseImageRef(ref string) (repo, tag string, err error) {
	repo, tag, hasTag := strings.Cut(ref, ":")
	if repo == "" {
		return "", "", fmt.Errorf("invalid image reference %q: missing repository", ref)
	}
	if hasTag && tag == "" {
		return "", "", fmt.Errorf("invalid image reference %q: empty tag", ref)
	}
	if !hasTag {
		tag = DefaultTag
	}
	return repo, tag, nil
}

// ArtifactNameFromProperties returns the non-empty artifact name from props.
func ArtifactNameFromProperties(props *properties.Properties) (string, error) {
	name, err := props.GetProperty(properties.PropertyName).AsString()
	if err != nil {
		return "", fmt.Errorf("failed to get artifact name: %w", err)
	}
	if name == "" {
		return "", errors.New("artifact name is empty")
	}
	return name, nil
}

// NewArtifactProperties builds the properties for a container artifact entity.
func NewArtifactProperties(name, upstreamID string) *properties.Properties {
	return properties.NewProperties(map[string]any{
		properties.PropertyName:         name,
		properties.PropertyUpstreamID:   upstreamID,
		properties.ArtifactPropertyType: string(verifyif.ArtifactTypeContainer),
	})
}
