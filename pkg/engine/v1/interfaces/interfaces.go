// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	Eval(ctx context.Context, profile map[string]any, entity protoreflect.ProtoMessage, res *Result) (*EvaluationResult, error)
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
	// BaseFs is the base filesystem for a pull request.  It can be used in the
	// evaluator for diffing the PR target files against the base files.
	BaseFs billy.Filesystem
	// Storer is the git storer that was created as a result of the ingestion.
	// FIXME: It might be cleaner to either wrap both Fs and Storer in a struct
	// or pass out the git.Repository structure instead of the storer.
	Storer storage.Storer

	// Checkpoint is the checkpoint at which the ingestion was done. This is
	// used to persist the state of the entity at ingestion time.
	Checkpoint *checkpoints.CheckpointEnvelopeV1
}

// EvaluationResult is the result of an evaluation
type EvaluationResult struct {
	// Output is the output of the evaluation. This contains a list of additional
	// information about the evaluation, which may be used in downstream actions.
	Output any
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
