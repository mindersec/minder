// Copyright 2023 Stacklok, Inc.
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
// Package rule provides the CLI subcommand for managing rules

// Package git provides the git rule data ingest engine
package git

import (
	"context"
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/db"
	engerrors "github.com/stacklok/minder/internal/engine/errors"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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
func NewGitIngester(cfg *pb.GitType, pbuild *providers.ProviderBuilder) (*Git, error) {
	if pbuild == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	if !pbuild.Implements(db.ProviderTypeGit) {
		return nil, fmt.Errorf("provider builder does not implement git")
	}

	if cfg == nil {
		cfg = &pb.GitType{}
	}

	gitprov, err := pbuild.GetGit()
	if err != nil {
		return nil, fmt.Errorf("could not get git provider: %w", err)
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
		}
		return nil, err
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("could not get worktree: %w", err)
	}

	return &engif.Result{
		Object: nil,
		Fs:     wt.Filesystem,
		Storer: r.Storer,
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
