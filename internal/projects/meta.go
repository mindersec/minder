//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package projects contains utilities for working with projects.
package projects

const (
	// MinderMetadataVersion is the version of the metadata format.
	MinderMetadataVersion = "v1alpha1"
)

// Metadata contains metadata relevant for a project.
type Metadata struct {
	Version      string `json:"version"`
	SelfEnrolled bool   `json:"self_enrolled"`
	Description  string `json:"description"`

	// TODO: Add more metadata fields here.
	// e.g. vendor-specific fields
}

// NewSelfEnrolledMetadata returns a new Metadata object with the SelfEnrolled field set to true.
func NewSelfEnrolledMetadata() Metadata {
	return Metadata{
		Version:      MinderMetadataVersion,
		SelfEnrolled: true,
	}
}
