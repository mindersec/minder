// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package git provides the git rule data ingest engine
package git

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"google.golang.org/protobuf/reflect/protoreflect"

	engerrors "github.com/mindersec/minder/internal/engine/errors"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/internal/entities/checkpoints"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// GitRuleDataIngestType is the type of the git rule data ingest engine
	GitRuleDataIngestType = "git"
	defaultBranch         = "main"
)

// Git is the engine for a rule type that uses git data ingest
type Git struct {
	cfg     *pb.GitType
	gitprov provifv1.Git
}

// NewGitIngester creates a new git rule data ingest engine
func NewGitIngester(cfg *pb.GitType, gitprov provifv1.Git) (*Git, error) {
	if gitprov == nil {
		return nil, fmt.Errorf("provider is nil")
	}

	if cfg == nil {
		cfg = &pb.GitType{}
	}

	return &Git{
		cfg:     cfg,
		gitprov: gitprov,
	}, nil
}

// GetType returns the type of the git rule data ingest engine
func (*Git) GetType() string {
	return GitRuleDataIngestType
}

// GetConfig returns the config for the git rule data ingest engine
func (gi *Git) GetConfig() protoreflect.ProtoMessage {
	return gi.cfg
}

// Ingest does the actual data ingestion for a rule type by cloning a git repo
func (gi *Git) Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*engif.Result, error) {
	userCfg := &IngesterConfig{}
	if err := mapstructure.Decode(params, userCfg); err != nil {
		return nil, fmt.Errorf("failed to read git ingester configuration from params: %w", err)
	}

	url := getCloneUrl(ent, userCfg)
	if url == "" {
		return nil, fmt.Errorf("could not get clone url")
	}

	branch := gi.getBranch(ent, userCfg)

	// We clone to the memfs go-billy filesystem driver, which doesn't
	// allow for direct access to the underlying filesystem. This is
	// because we want to be able to run this in a sandboxed environment
	// where we don't have access to the underlying filesystem.
	r, err := gi.gitprov.Clone(ctx, url, branch)
	if err != nil {
		if errors.Is(err, provifv1.ErrProviderGitBranchNotFound) {
			return nil, fmt.Errorf("%w: %s: branch %s", engerrors.ErrEvaluationFailed,
				provifv1.ErrProviderGitBranchNotFound, branch)
		} else if errors.Is(err, provifv1.ErrRepositoryEmpty) {
			return nil, fmt.Errorf("%w: %s", engerrors.ErrEvaluationSkipped, provifv1.ErrRepositoryEmpty)
		}
		return nil, err
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("could not get worktree: %w", err)
	}

	head, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("could not get head: %w", err)
	}

	hsh := head.Hash()

	chkpoint := checkpoints.NewCheckpointV1Now().
		WithBranch(branch).
		WithCommitHash(hsh.String())

	return &engif.Result{
		Object:     nil,
		Fs:         wt.Filesystem,
		Storer:     r.Storer,
		Checkpoint: chkpoint,
	}, nil
}

func (gi *Git) getBranch(ent protoreflect.ProtoMessage, userCfg *IngesterConfig) string {
	// If the user has specified a branch, use that
	if userCfg.Branch != "" {
		return userCfg.Branch
	}

	// If the branch is provided in the rule-type
	// configuration, use that
	if gi.cfg.Branch != "" {
		return gi.cfg.Branch
	}

	// If the entity is a repository get it from the entity
	// else, use the default
	if repo, ok := ent.(*pb.Repository); ok {
		if repo.GetDefaultBranch() != "" {
			return repo.GetDefaultBranch()
		}
	}

	// If the branch is not provided in the rule-type
	// configuration, use the default branch
	return defaultBranch
}

func getCloneUrl(ent protoreflect.ProtoMessage, cfg *IngesterConfig) string {
	if cfg.CloneURL != "" {
		return cfg.CloneURL
	}

	// If the entity is a repository get it from the entity
	// else, get it from the configuration
	if repo, ok := ent.(*pb.Repository); ok {
		return repo.GetCloneUrl()
	}

	return ""
}
