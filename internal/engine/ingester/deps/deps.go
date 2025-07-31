// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package deps provides the deps rule data ingest engine
package deps

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/go-git/go-billy/v5/helper/iofs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-viper/mapstructure/v2"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	mdeps "github.com/mindersec/minder/internal/deps"
	"github.com/mindersec/minder/internal/deps/scalibr"
	pbinternal "github.com/mindersec/minder/internal/proto"
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
	gitprov   interfaces.GitProvider
	extractor mdeps.Extractor
}

// RepoConfig is the set of parameters to the deps rule data ingest engine for repositories
type RepoConfig struct {
	Branch string `json:"branch" yaml:"branch" mapstructure:"branch"`
}

// PullRequestConfig is the set of parameters to the deps rule data ingest engine for pull requests
type PullRequestConfig struct {
	Filter string `json:"filter" yaml:"filter" mapstructure:"filter"`
}

const (
	// PullRequestIngestTypeNew is a filter that exposes only new dependencies in the pull request
	PullRequestIngestTypeNew = "new"
	// PullRequestIngestTypeNewAndUpdated is a filter that exposes new and updated
	// dependencies in the pull request
	PullRequestIngestTypeNewAndUpdated = "new_and_updated"
	// PullRequestIngestTypeAll is a filter that exposes all dependencies in the pull request
	PullRequestIngestTypeAll = "all"
)

// NewDepsIngester creates a new deps rule data ingest engine
func NewDepsIngester(cfg *pb.DepsType, gitprov interfaces.GitProvider) (*Deps, error) {
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
func (gi *Deps) Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*interfaces.Ingested, error) {
	switch entity := ent.(type) {
	case *pb.Repository:
		return gi.ingestRepository(ctx, entity, params)
	case *pbinternal.PullRequest:
		return gi.ingestPullRequest(ctx, entity, params)
	default:
		return nil, fmt.Errorf("deps is only supported for repositories and pull requests")
	}
}

func (gi *Deps) ingestRepository(ctx context.Context, repo *pb.Repository, params map[string]any) (*interfaces.Ingested, error) {
	var logger = zerolog.Ctx(ctx)
	// the branch is left unset since we want to auto-discover it
	// in case it's not explicitly set
	userCfg := &RepoConfig{}
	if err := mapstructure.Decode(params, userCfg); err != nil {
		return nil, fmt.Errorf("failed to read dependency ingester configuration from params: %w", err)
	}

	if repo.GetCloneUrl() == "" {
		return nil, fmt.Errorf("could not get clone url")
	}

	branch := gi.getBranch(repo, userCfg.Branch)
	logger.Info().Interface("repo", repo).Msgf("extracting dependencies from %s#%s", repo.GetCloneUrl(), branch)

	deps, head, err := gi.scanFromUrl(ctx, repo.GetCloneUrl(), branch)
	if err != nil {
		return nil, fmt.Errorf("could not scan filesystem: %w", err)
	}

	logger.Debug().Interface("deps", deps).Msgf("Scanning successful: %d nodes found", len(deps.Nodes))

	hsh := head.Hash()

	chkpoint := checkpoints.NewCheckpointV1Now().
		WithBranch(branch).
		WithCommitHash(hsh.String())

	return &interfaces.Ingested{
		Object: map[string]any{
			"node_list": deps,
		},
		Checkpoint: chkpoint,
	}, nil
}

func (gi *Deps) getBranch(repo *pb.Repository, userConfigBranch string) string {
	// If the user has specified a branch, use that
	if userConfigBranch != "" {
		return userConfigBranch
	}

	// If the branch is provided in the rule-type
	// configuration, use that
	if gi.cfg.GetRepo().GetBranch() != "" {
		return gi.cfg.GetRepo().GetBranch()
	}
	if repo.GetDefaultBranch() != "" {
		return repo.GetDefaultBranch()
	}

	// If the branch is not provided in the rule-type
	// configuration, use the default branch
	return defaultBranch
}

// ingestTypes returns a sorter function for the given filter type.
// items which compare equal are skipped in output.
var ingestTypes = map[string]func(*sbom.Node, *sbom.Node) int{
	PullRequestIngestTypeNew: func(base *sbom.Node, updated *sbom.Node) int {
		return cmp.Compare(base.GetName(), updated.GetName())
	},
	PullRequestIngestTypeNewAndUpdated: func(base *sbom.Node, updated *sbom.Node) int {
		return nodeSorter(base, updated)
	},
	PullRequestIngestTypeAll: func(_ *sbom.Node, _ *sbom.Node) int {
		return -1
	},
}

func nodeSorter(a *sbom.Node, b *sbom.Node) int {
	// If we compare by name and version first, we can avoid computing map keys.
	res := cmp.Or(cmp.Compare(a.GetName(), b.GetName()),
		cmp.Compare(a.GetVersion(), b.GetVersion()))
	if res != 0 {
		return res
	}
	// Same name and version, compare hashes.  Go's shuffling map keys does not help here.
	aHashes := make([]int32, 0, len(a.GetHashes()))
	for algo := range a.GetHashes() {
		aHashes = append(aHashes, algo)
	}
	slices.Sort(aHashes)
	bHashes := make([]int32, 0, len(b.GetHashes()))
	for algo := range b.GetHashes() {
		bHashes = append(bHashes, algo)
	}
	slices.Sort(bHashes)
	for i, algo := range aHashes {
		if i >= len(bHashes) {
			return 1
		}
		if r := cmp.Compare(algo, bHashes[i]); r != 0 {
			return r
		}
		if r := cmp.Compare(a.GetHashes()[algo], b.GetHashes()[algo]); r != 0 {
			return r
		}
	}
	if len(aHashes) < len(bHashes) {
		return -1
	}
	return 0
}

func filterNodes(base []*sbom.Node, updated []*sbom.Node, compare func(*sbom.Node, *sbom.Node) int) []*sbom.Node {
	slices.SortFunc(base, nodeSorter)
	slices.SortFunc(updated, nodeSorter)

	ret := make([]*sbom.Node, 0, len(updated))

	baseIdx, newIdx := 0, 0
	for baseIdx < len(base) && newIdx < len(updated) {
		cmpResult := compare(base[baseIdx], updated[newIdx])
		if cmpResult < 0 {
			baseIdx++
		} else if cmpResult > 0 {
			ret = append(ret, updated[newIdx])
			newIdx++
		} else {
			newIdx++
		}
	}
	if newIdx < len(updated) {
		ret = append(ret, updated[newIdx:]...)
	}
	return ret
}

func (gi *Deps) ingestPullRequest(
	ctx context.Context, pr *pbinternal.PullRequest, params map[string]any) (*interfaces.Ingested, error) {
	userCfg := &PullRequestConfig{
		// We default to new_and_updated for user convenience.
		Filter: PullRequestIngestTypeNewAndUpdated,
	}
	if err := mapstructure.Decode(params, userCfg); err != nil {
		return nil, fmt.Errorf("failed to read dependency ingester configuration from params: %w", err)
	}

	// Enforce that the filter is valid if left empty.
	if userCfg.Filter == "" {
		userCfg.Filter = PullRequestIngestTypeNewAndUpdated
	}

	// At this point the user really set a wrong configuration. So, let's error out.
	if _, ok := ingestTypes[userCfg.Filter]; !ok {
		return nil, fmt.Errorf("invalid filter type: %s", userCfg.Filter)
	}

	if pr.GetBaseCloneUrl() == "" {
		return nil, errors.New("could not get base clone url")
	}
	if pr.GetTargetCloneUrl() == "" {
		return nil, errors.New("could not get head clone url")
	}
	baseDeps, _, err := gi.scanFromUrl(ctx, pr.GetBaseCloneUrl(), pr.GetBaseRef())
	if err != nil {
		return nil, fmt.Errorf("could not scan base filesystem: %w", err)
	}
	targetDeps, ref, err := gi.scanFromUrl(ctx, pr.GetTargetCloneUrl(), pr.GetTargetRef())
	if err != nil {
		return nil, fmt.Errorf("could not scan target filesystem: %w", err)
	}

	// Overwrite the target list of nodes with the result of filtering by desired match.
	// We checked that the filter is valid at the top of the function.
	targetDeps.Nodes = filterNodes(baseDeps.GetNodes(), targetDeps.GetNodes(), ingestTypes[userCfg.Filter])

	chkpoint := checkpoints.NewCheckpointV1Now().
		WithBranch(pr.GetTargetRef()).
		WithCommitHash(ref.Hash().String())

	return &interfaces.Ingested{
		Object: map[string]any{
			"node_list": targetDeps,
		},
		Checkpoint: chkpoint,
	}, nil
}

// TODO: this first part is fairly shared with fetchClone from ../git/git.go.
func (gi *Deps) scanFromUrl(ctx context.Context, url string, branch string) (*sbom.NodeList, *plumbing.Reference, error) {
	// We clone to the memfs go-billy filesystem driver, which doesn't
	// allow for direct access to the underlying filesystem. This is
	// because we want to be able to run this in a sandboxed environment
	// where we don't have access to the underlying filesystem.
	repo, err := gi.gitprov.Clone(ctx, url, branch)
	if err != nil {
		if errors.Is(err, provifv1.ErrProviderGitBranchNotFound) {
			return nil, nil, fmt.Errorf("%w: %s: branch %s", interfaces.ErrEvaluationFailed,
				provifv1.ErrProviderGitBranchNotFound, branch)
		} else if errors.Is(err, provifv1.ErrRepositoryEmpty) {
			return nil, nil, fmt.Errorf("%w: %s", interfaces.ErrEvaluationSkipped, provifv1.ErrRepositoryEmpty)
		}
		return nil, nil, err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get worktree: %w", err)
	}

	if wt.Filesystem == nil {
		return nil, nil, fmt.Errorf("could not get filesystem")
	}

	deps, err := gi.extractor.ScanFilesystem(ctx, iofs.New(wt.Filesystem))
	if err != nil {
		return nil, nil, fmt.Errorf("%T extractor: %w", gi.extractor, err)
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get head: %w", err)
	}

	return deps, ref, nil
}
