// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package deps provides the deps rule data ingest engine
package deps

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/iofs"
	"github.com/go-viper/mapstructure/v2"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	mdeps "github.com/mindersec/minder/internal/deps"
	"github.com/mindersec/minder/internal/deps/scalibr"
	engerrors "github.com/mindersec/minder/internal/engine/errors"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/entities/v1/checkpoints"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// DepsRuleDataIngestType is the type of the deps rule data ingest engine
	DepsRuleDataIngestType = "deps"
	defaultBranch          = "main"
)

// Deps is the engine for a rule type that uses deps data ingest
type Deps struct {
	cfg       *pb.DepsType
	gitprov   provifv1.Git
	extractor mdeps.Extractor
}

// Config is the set of parameters to the deps rule data ingest engine
type Config struct {
	Branch string `json:"branch" yaml:"branch" mapstructure:"branch"`
}

// NewDepsIngester creates a new deps rule data ingest engine
func NewDepsIngester(cfg *pb.DepsType, gitprov provifv1.Git) (*Deps, error) {
	if gitprov == nil {
		return nil, fmt.Errorf("provider is nil")
	}

	if cfg == nil {
		cfg = &pb.DepsType{}
	}

	return &Deps{
		cfg:       cfg,
		gitprov:   gitprov,
		extractor: scalibr.NewExtractor(),
	}, nil
}

// GetType returns the type of the git rule data ingest engine
func (*Deps) GetType() string {
	return DepsRuleDataIngestType
}

// GetConfig returns the config for the git rule data ingest engine
func (gi *Deps) GetConfig() protoreflect.ProtoMessage {
	return gi.cfg
}

// Ingest does the actual data ingestion for a rule type by cloning a git repo,
// and scanning it for dependencies with a dependency extractor
func (gi *Deps) Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*interfaces.Result, error) {
	switch entity := ent.(type) {
	case *pb.Repository:
		return gi.ingestRepository(ctx, entity, params)
	default:
		return nil, fmt.Errorf("deps is only supported for repositories")
	}
}

func (gi *Deps) ingestRepository(ctx context.Context, repo *pb.Repository, params map[string]any) (*interfaces.Result, error) {
	var logger = zerolog.Ctx(ctx)
	userCfg := &Config{
		Branch: defaultBranch,
	}
	if err := mapstructure.Decode(params, userCfg); err != nil {
		return nil, fmt.Errorf("failed to read dependency ingester configuration from params: %w", err)
	}

	if repo.GetCloneUrl() == "" {
		return nil, fmt.Errorf("could not get clone url")
	}

	branch := gi.getBranch(repo, userCfg.Branch)
	logger.Info().Interface("repo", repo).Msgf("extracting dependencies from %s#%s", repo.GetCloneUrl(), branch)

	// We clone to the memfs go-billy filesystem driver, which doesn't
	// allow for direct access to the underlying filesystem. This is
	// because we want to be able to run this in a sandboxed environment
	// where we don't have access to the underlying filesystem.
	r, err := gi.gitprov.Clone(ctx, repo.GetCloneUrl(), branch)
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

	deps, err := gi.scanMemFs(ctx, wt.Filesystem)
	if err != nil {
		return nil, fmt.Errorf("could not scan filesystem: %w", err)
	}

	logger.Debug().Interface("deps", deps).Msgf("Scanning successful: %d nodes found", len(deps.Nodes))

	head, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("could not get head: %w", err)
	}

	hsh := head.Hash()

	chkpoint := checkpoints.NewCheckpointV1Now().
		WithBranch(branch).
		WithCommitHash(hsh.String())

	return &interfaces.Result{
		Object: map[string]any{
			"node_list": deps,
		},
		Checkpoint: chkpoint,
	}, nil
}

func (gi *Deps) getBranch(repo *pb.Repository, branch string) string {
	// If the user has specified a branch, use that
	if branch != "" {
		return branch
	}

	// If the branch is provided in the rule-type
	// configuration, use that
	if gi.cfg.GetRepo().Branch != "" {
		return gi.cfg.GetRepo().Branch
	}
	if repo.GetDefaultBranch() != "" {
		return repo.GetDefaultBranch()
	}

	// If the branch is not provided in the rule-type
	// configuration, use the default branch
	return defaultBranch
}

// scanMemFs scans a billy memory filesystem for software dependencies.
func (gi *Deps) scanMemFs(ctx context.Context, memFS billy.Filesystem) (*sbom.NodeList, error) {
	if memFS == nil {
		return nil, fmt.Errorf("unable to scan dependencies, no active defined")
	}

	nl, err := gi.extractor.ScanFilesystem(ctx, iofs.New(memFS))
	if err != nil {
		return nil, fmt.Errorf("%T extractor: %w", gi.extractor, err)
	}

	return nl, err
}
