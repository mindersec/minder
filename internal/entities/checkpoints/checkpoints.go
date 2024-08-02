// Copyright 2024 Stacklok, Inc.
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

// Package checkpoints contains logic relating to checkpoint management for entities
package checkpoints

import "time"

// V1 is the version string for the v1 format.
const V1 = "v1"

// CheckpointEnvelopeV1 is the top-level structure for a checkpoint
// in the v1 format.
type CheckpointEnvelopeV1 struct {
	Version    string       `json:"version" yaml:"version"`
	Checkpoint CheckpointV1 `json:"checkpoint" yaml:"checkpoint"`
}

// CheckpointV1 is the structure for a checkpoint in the v1 format.
type CheckpointV1 struct {
	// Timestamp is the time that the checkpoint was verified.
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// CommitHash is the hash of the commit that the checkpoint is for.
	CommitHash *string `json:"commitHash,omitempty" yaml:"commitHash,omitempty"`

	// Branch is the branch of the commit that the checkpoint is for.
	Branch *string `json:"branch,omitempty" yaml:"branch,omitempty"`

	// Version is the version of the entity that the checkpoint is for.
	// This may be a container image tag, a git tag, or some other version.
	Version *string `json:"version,omitempty" yaml:"version,omitempty"`

	// Digest is the digest of the entity that the checkpoint is for.
	// This may be a container image digest, or some other digest.
	Digest *string `json:"digest,omitempty" yaml:"digest,omitempty"`
}

// NewCheckpointV1 creates a new CheckpointV1 with the given timestamp.
func NewCheckpointV1(timestamp time.Time) *CheckpointEnvelopeV1 {
	return &CheckpointEnvelopeV1{
		Version: V1,
		Checkpoint: CheckpointV1{
			Timestamp: timestamp,
		},
	}
}

// WithCommitHash sets the commit hash on the checkpoint.
func (c *CheckpointEnvelopeV1) WithCommitHash(commitHash string) *CheckpointEnvelopeV1 {
	c.Checkpoint.CommitHash = &commitHash
	return c
}

// WithBranch sets the branch on the checkpoint.
func (c *CheckpointEnvelopeV1) WithBranch(branch string) *CheckpointEnvelopeV1 {
	c.Checkpoint.Branch = &branch
	return c
}

// WithVersion sets the version on the checkpoint.
func (c *CheckpointEnvelopeV1) WithVersion(version string) *CheckpointEnvelopeV1 {
	c.Checkpoint.Version = &version
	return c
}

// WithDigest sets the digest on the checkpoint.
func (c *CheckpointEnvelopeV1) WithDigest(digest string) *CheckpointEnvelopeV1 {
	c.Checkpoint.Digest = &digest
	return c
}
