// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package projects contains utilities for working with projects.
package projects

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mindersec/minder/internal/db"
)

const (
	// MinderMetadataVersion is the version of the metadata format.
	MinderMetadataVersion = "v1alpha1"
)

var (
	// ErrValidationFailed is returned when a project fails validation
	ErrValidationFailed = fmt.Errorf("validation failed")

	// Patterns for validating project names
	projectAlnum     = regexp.MustCompile(`^[a-zA-Z0-9](?:[-_a-zA-Z0-9]{0,61}[a-zA-Z0-9])?$`)
	projectUuidAlike = regexp.MustCompile(`^[[:alnum:]]{8}-([[:alnum:]]{4}-){3}[[:alnum:]]{12}$`)
	projectDnsName   = regexp.MustCompile(`^(?:(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])$`)
)

// Metadata contains metadata relevant for a project.
type Metadata struct {
	Version      string `json:"version"`
	SelfEnrolled bool   `json:"self_enrolled"`

	// This will be deprecated in favor of PublicMetadataV1.
	Description string `json:"description"`

	// TODO: Add more metadata fields here.
	// e.g. vendor-specific fields

	// Public is a field that is meant to be read by other systems.
	// It will be exposed to the public, e.g. via a UI.
	Public PublicMetadataV1 `json:"public"`
}

// PublicMetadataV1 contains public metadata relevant for a project.
type PublicMetadataV1 struct {
	Description string `json:"description"`
	DisplayName string `json:"display_name"`
}

// NewSelfEnrolledMetadata returns a new Metadata object with the SelfEnrolled field set to true.
func NewSelfEnrolledMetadata(projectName string) Metadata {
	return Metadata{
		Version:      MinderMetadataVersion,
		SelfEnrolled: true,
		// These will be editable by the user.
		Public: PublicMetadataV1{
			Description: "A self-enrolled project.",
			DisplayName: projectName,
		},
	}
}

// ParseMetadata parses the given JSON data into a Metadata object.
func ParseMetadata(proj *db.Project) (*Metadata, error) {
	var meta Metadata
	if err := json.Unmarshal(proj.Metadata, &meta); err != nil {
		return nil, err
	}

	// default the display name to the project name if it's not set
	if meta.Public.DisplayName == "" {
		meta.Public.DisplayName = proj.Name
	}

	return &meta, nil
}

// ValidateName validates the given project name.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrValidationFailed)
	}

	if strings.Contains(name, "/") {
		return fmt.Errorf("%w: name cannot contain '/'", ErrValidationFailed)
	}

	// Don't allow names that look like UUIDs, because that is confusing.
	// Note that we are somewhat loose here, because l/1 and other look-alike
	// characters in a 36-character string.
	if projectUuidAlike.MatchString(name) {
		return fmt.Errorf("%w: name cannot be UUID-like", ErrValidationFailed)
	}

	// Check if the name is too long.
	if len(name) > 63 {
		return fmt.Errorf("%w: name is too long", ErrValidationFailed)
	}

	if !projectAlnum.MatchString(name) && !projectDnsName.MatchString(name) {
		return fmt.Errorf("%w: name must be a valid DNS name or an alphanumeric sequence", ErrValidationFailed)
	}

	return nil
}

// SerializeMetadata serializes the given Metadata object into JSON.
func SerializeMetadata(meta *Metadata) ([]byte, error) {
	return json.Marshal(meta)
}
