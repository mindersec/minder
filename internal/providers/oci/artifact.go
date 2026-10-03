// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mindersec/minder/internal/verifier/verifyif"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

// ArtifactV1FromProperties converts container artifact properties to a minder v1 Artifact.
// Owner is the provider namespace and Name is the repository without its tag, so the
// artifact resolves to <registry>/<namespace>/<repo>.
func ArtifactV1FromProperties(props *properties.Properties, namespace string) (*minderv1.Artifact, error) {
	name, err := ArtifactNameFromProperties(props)
	if err != nil {
		return nil, err
	}

	repo, _, err := ParseImageRef(name)
	if err != nil {
		return nil, err
	}

	upstreamID, err := props.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact upstream ID: %w", err)
	}

	// An empty namespace would build "<registry>//<repo>" refs.
	if namespace == "" {
		return nil, errors.New("artifact namespace is empty")
	}

	return &minderv1.Artifact{
		ArtifactPk: upstreamID,
		Owner:      namespace,
		Name:       repo,
		Type:       string(verifyif.ArtifactTypeContainer),
	}, nil
}
