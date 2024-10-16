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

// Package interfaces contains the interfaces for the Minder policy engine.
package interfaces

import (
	"context"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/storage"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/pkg/entities/v1/checkpoints"
)

// Ingester is the interface for a rule type ingester
type Ingester interface {
	// Ingest does the actual data ingestion for a rule type
	Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*Result, error)
	// GetType returns the type of the ingester
	GetType() string
	// GetConfig returns the config for the ingester
	GetConfig() protoreflect.ProtoMessage
}

// Evaluator is the interface for a rule type evaluator
type Evaluator interface {
	Eval(ctx context.Context, profile map[string]any, entity protoreflect.ProtoMessage, res *Result) error
}

// Result is the result of an ingester
type Result struct {
	// Object is the object that was ingested. Normally comes from an external
	// system like an HTTP server.
	Object any
	// Fs is the filesystem that was created as a result of the ingestion. This
	// is normally used by the evaluator to do rule evaluation. The filesystem
	// may be a git repo, or a memory filesystem.
	Fs billy.Filesystem
	// Storer is the git storer that was created as a result of the ingestion.
	// FIXME: It might be cleaner to either wrap both Fs and Storer in a struct
	// or pass out the git.Repository structure instead of the storer.
	Storer storage.Storer

	// Checkpoint is the checkpoint at which the ingestion was done. This is
	// used to persist the state of the entity at ingestion time.
	Checkpoint *checkpoints.CheckpointEnvelopeV1
}

// GetCheckpoint returns the checkpoint of the result
func (r *Result) GetCheckpoint() *checkpoints.CheckpointEnvelopeV1 {
	if r == nil {
		return nil
	}

	return r.Checkpoint
}

// ResultSink sets the result of an ingestion
type ResultSink interface {
	SetIngestResult(*Result)
}
