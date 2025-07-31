// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package git provides the git rule data ingest engine
package git

import (
	"cmp"
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-viper/mapstructure/v2"
	"google.golang.org/protobuf/reflect/protoreflect"

	pbinternal "github.com/mindersec/minder/internal/proto"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/entities/v1/checkpoints"
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
	gitprov interfaces.GitProvider
}

// NewGitIngester creates a new git rule data ingest engine
func NewGitIngester(cfg *pb.GitType, gitprov interfaces.GitProvider) (*Git, error) {
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
func (gi *Git) Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*interfaces.Ingested, error) {
	switch entity := ent.(type) {
	case *pb.Repository:
		return gi.ingestRepository(ctx, entity, params)
	case *pbinternal.PullRequest:
		return gi.ingestPullRequest(ctx, entity, params)
	default:
		return nil, fmt.Errorf("git is only supported for repositories and pull requests")
	}
}

func (gi *Git) ingestRepository(ctx context.Context, repo *pb.Repository, params map[string]any) (*interfaces.Ingested, error) {
	userCfg := &IngesterConfig{}
	if err := mapstructure.Decode(params, userCfg); err != nil {
		return nil, fmt.Errorf("failed to read git ingester configuration from params: %w", err)
	}

	url := cmp.Or(userCfg.CloneURL, repo.GetCloneUrl())
	if url == "" {
		return nil, fmt.Errorf("could not get clone url")
	}

	branch := cmp.Or(userCfg.Branch, gi.cfg.Branch, repo.GetDefaultBranch(), defaultBranch)
	fs, storer, head, err := gi.fetchClone(ctx, url, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to clone %s from %s: %w", branch, url, err)
	}

	hsh := head.Hash()

	chkpoint := checkpoints.NewCheckpointV1Now().
		WithBranch(branch).
		WithCommitHash(hsh.String())

	return &interfaces.Ingested{
		Object:     nil,
		Fs:         fs,
		Storer:     storer,
		Checkpoint: chkpoint,
	}, nil
}

func (gi *Git) ingestPullRequest(
	ctx context.Context, ent *pbinternal.PullRequest, params map[string]any) (*interfaces.Ingested, error) {
	// TODO: we don't actually have any configuration here.  Do we need to read the configuration?
	userCfg := &IngesterConfig{}
	if err := mapstructure.Decode(params, userCfg); err != nil {
		return nil, fmt.Errorf("failed to read git ingester configuration from params: %w", err)
	}

	if ent.GetBaseCloneUrl() == "" || ent.GetBaseRef() == "" {
		return nil, fmt.Errorf("could not get PR base branch %q from %q", ent.GetBaseRef(), ent.GetBaseCloneUrl())
	}
	if ent.GetTargetCloneUrl() == "" || ent.GetTargetRef() == "" {
		return nil, fmt.Errorf("could not get PR target branch %q from %q", ent.GetTargetRef(), ent.GetTargetCloneUrl())
	}

	baseFs, _, _, err := gi.fetchClone(ctx, ent.GetBaseCloneUrl(), ent.GetBaseRef())
	if err != nil {
		return nil, fmt.Errorf("failed to clone base branch %s from %s: %w", ent.GetBaseRef(), ent.GetBaseCloneUrl(), err)
	}
	targetFs, storer, head, err := gi.fetchClone(ctx, ent.GetTargetCloneUrl(), ent.GetTargetRef())
	if err != nil {
		return nil, fmt.Errorf("failed to clone target branch %s from %s: %w", ent.GetTargetRef(), ent.GetTargetCloneUrl(), err)
	}

	checkpoint := checkpoints.NewCheckpointV1Now().WithBranch(ent.GetTargetRef()).WithCommitHash(head.Hash().String())

	return &interfaces.Ingested{
		Object:     nil,
		Fs:         targetFs,
		Storer:     storer,
		BaseFs:     baseFs,
		Checkpoint: checkpoint,
	}, nil
}

func (gi *Git) fetchClone(
	ctx context.Context, url, branch string) (billy.Filesystem, storage.Storer, *plumbing.Reference, error) {
	// We clone to the memfs go-billy filesystem driver, which doesn't
	// allow for direct access to the underlying filesystem. This is
	// because we want to be able to run this in a sandboxed environment
	// where we don't have access to the underlying filesystem.
	r, err := gi.gitprov.Clone(ctx, url, branch)
	if err != nil {
		if errors.Is(err, provifv1.ErrProviderGitBranchNotFound) {
			return nil, nil, nil, fmt.Errorf("%w: %s: branch %s", interfaces.ErrEvaluationFailed,
				provifv1.ErrProviderGitBranchNotFound, branch)
		} else if errors.Is(err, provifv1.ErrRepositoryEmpty) {
			return nil, nil, nil, fmt.Errorf("%w: %s", interfaces.ErrEvaluationSkipped, provifv1.ErrRepositoryEmpty)
		}
		return nil, nil, nil, err
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not get worktree: %w", err)
	}

	head, err := r.Head()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not get head: %w", err)
	}

	return wt.Filesystem, r.Storer, head, err
}
