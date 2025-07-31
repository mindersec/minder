// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package interfaces contains the interfaces for the Minder policy engine.
package interfaces

import (
	"context"
	"errors"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/storage"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/pkg/entities/v1/checkpoints"
)

// Ingester is the interface for a rule type ingester
type Ingester interface {
	// Ingest does the actual data ingestion for a rule type
	Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*Ingested, error)
	// GetType returns the type of the ingester
	GetType() string
	// GetConfig returns the config for the ingester
	GetConfig() protoreflect.ProtoMessage
}

// Evaluator is the interface for a rule type evaluator
//
// `profile` is a set of parameters exposed to the rule evaluation by the rule engine
// `entity` is one of minderv1.Repository or minderv1.Artifact
// `data` is the data ingested
type Evaluator interface {
	Eval(ctx context.Context, profile map[string]any, entity protoreflect.ProtoMessage, data *Ingested) (*EvaluationResult, error)
}

// Option is a function that takes an evaluator and does some
// unspecified operation to it, returning an error in case of failure.
type Option func(Evaluator) error

// EvalError is an interface providing additional details from Evaluator.Eval()
// errors when the evaluation determines that the rule is violated.
type EvalError interface {
	Error() string
	Details() string
}

// ErrEvaluationFailed is an error that occurs during evaluation of a rule.
var ErrEvaluationFailed = errors.New("evaluation failure")

// ErrEvaluationSkipped specifies that the rule was evaluated but skipped.
var ErrEvaluationSkipped = errors.New("evaluation skipped")

// Ingested is the result of an ingester
type Ingested struct {
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
func (r *Ingested) GetCheckpoint() *checkpoints.CheckpointEnvelopeV1 {
	if r == nil {
		return nil
	}

	return r.Checkpoint
}

// ResultSink sets the result of an ingestion
type ResultSink interface {
	SetIngestResult(*Ingested)
}
