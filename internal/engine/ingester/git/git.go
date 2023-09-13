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
	"fmt"

	memfs "github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	http "github.com/go-git/go-git/v5/plumbing/transport/http"
	memory "github.com/go-git/go-git/v5/storage/memory"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const (
	// GitRuleDataIngestType is the type of the git rule data ingest engine
	GitRuleDataIngestType = "git"
	defaultBranch         = "main"
)

// Git is the engine for a rule type that uses git data ingest
type Git struct {
	accessToken string
	cfg         *pb.GitType
}

// NewGitIngester creates a new git rule data ingest engine
func NewGitIngester(cfg *pb.GitType, token string) *Git {
	if cfg == nil {
		cfg = &pb.GitType{}
	}
	return &Git{
		accessToken: token,
		cfg:         cfg,
	}
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

	branch := gi.getBranch(userCfg)

	opts := &git.CloneOptions{
		URL:           url,
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.NoTags,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}

	if gi.accessToken != "" {
		opts.Auth = &http.BasicAuth{
			// the Username can be anything but it can't be empty
			Username: "mediator-user",
			Password: gi.accessToken,
		}
	}

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid clone options: %w", err)
	}

	storer := memory.NewStorage()
	fs := memfs.New()

	// We clone to the memfs go-billy filesystem driver, which doesn't
	// allow for direct access to the underlying filesystem. This is
	// because we want to be able to run this in a sandboxed environment
	// where we don't have access to the underlying filesystem.
	r, err := git.CloneContext(ctx, storer, fs, opts)
	if err != nil {
		return nil, fmt.Errorf("could not clone repo: %w", err)
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("could not get worktree: %w", err)
	}

	return &engif.Result{
		Object: nil,
		Fs:     wt.Filesystem,
	}, nil
}

func (gi *Git) getBranch(userCfg *IngesterConfig) string {
	// If the user has specified a branch, use that
	if userCfg.Branch != "" {
		return userCfg.Branch
	}

	// If the branch is provided in the rule-type
	// configuration, use that
	if gi.cfg.Branch != "" {
		return gi.cfg.Branch
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
	if repo, ok := ent.(*pb.RepositoryResult); ok {
		return repo.GetCloneUrl()
	}

	return ""
}
