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
	scalibr "github.com/google/osv-scalibr"
	"github.com/google/osv-scalibr/extractor/filesystem/list"
	scalibr_fs "github.com/google/osv-scalibr/fs"
	scalibr_plugin "github.com/google/osv-scalibr/plugin"
	"github.com/google/uuid"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

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
	cfg     *pb.DepsType
	gitprov provifv1.Git
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
		cfg:     cfg,
		gitprov: gitprov,
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
// and scanning it for dependencies with scalibr.
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

	deps, err := scanFs(ctx, wt.Filesystem)
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

func scanFs(ctx context.Context, memFS billy.Filesystem) (*sbom.NodeList, error) {
	if memFS == nil {
		return nil, fmt.Errorf("unable to scan dependencies, no active defined")
	}
	// have to down-cast here, because scalibr needs multiple io/fs types
	wrapped, ok := iofs.New(memFS).(scalibr_fs.FS)
	if !ok {
		return nil, fmt.Errorf("error converting filesystem to ReadDirFS")
	}

	desiredCaps := scalibr_plugin.Capabilities{
		OS:            scalibr_plugin.OSLinux,
		Network:       true,
		DirectFS:      false,
		RunningSystem: false,
	}

	scalibrFs := scalibr_fs.ScanRoot{FS: wrapped}
	scanConfig := scalibr.ScanConfig{
		ScanRoots: []*scalibr_fs.ScanRoot{&scalibrFs},
		// All includes Ruby, Dotnet which we're not ready to test yet, so use the more limited Default set.
		FilesystemExtractors: list.FilterByCapabilities(list.Default, &desiredCaps),
		Capabilities:         &desiredCaps,
	}

	scanner := scalibr.New()
	scanResults := scanner.Scan(ctx, &scanConfig)

	if scanResults == nil || scanResults.Status == nil {
		return nil, fmt.Errorf("error scanning files: no results")
	}
	if scanResults.Status.Status != scalibr_plugin.ScanStatusSucceeded {
		return nil, fmt.Errorf("error scanning files: %s", scanResults.Status)
	}

	res := sbom.NewNodeList()
	for _, inv := range scanResults.Inventories {
		node := &sbom.Node{
			Type:    sbom.Node_PACKAGE,
			Id:      uuid.New().String(),
			Name:    inv.Name,
			Version: inv.Version,
			Identifiers: map[int32]string{
				int32(sbom.SoftwareIdentifierType_PURL): inv.Extractor.ToPURL(inv).String(),
				// TODO: scalibr returns a _list_ of CPEs, but protobom will store one.
				// use the first?
				// int32(sbom.SoftwareIdentifierType_CPE23):  inv.Extractor.ToCPEs(inv),
			},
		}
		for _, l := range inv.Locations {
			node.Properties = append(node.Properties, &sbom.Property{
				Name: "sourceFile",
				Data: l,
			})
		}
		res.AddNode(node)
	}

	return res, nil
}
