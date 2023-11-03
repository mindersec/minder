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

// Package diff provides the diff rule data ingest engine
package diff

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// DiffRuleDataIngestType is the type of the diff rule data ingest engine
	DiffRuleDataIngestType = "diff"
	prFilesPerPage         = 30
	wildcard               = "*"
)

// Diff is the diff rule data ingest engine
type Diff struct {
	cli provifv1.GitHub
	cfg *pb.DiffType
}

// NewDiffIngester creates a new diff ingester
func NewDiffIngester(
	cfg *pb.DiffType,
	pbuild *providers.ProviderBuilder,
) (*Diff, error) {
	if cfg == nil {
		cfg = &pb.DiffType{}
	}

	if pbuild == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	cli, err := pbuild.GetGitHub(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get github client: %w", err)
	}

	return &Diff{
		cfg: cfg,
		cli: cli,
	}, nil
}

// GetType returns the type of the diff rule data ingest engine
func (*Diff) GetType() string {
	return DiffRuleDataIngestType
}

// GetConfig returns the config for the diff rule data ingest engine
func (di *Diff) GetConfig() protoreflect.ProtoMessage {
	return di.cfg
}

// Ingest ingests a pull request and returns a list of dependencies
func (di *Diff) Ingest(
	ctx context.Context,
	ent protoreflect.ProtoMessage,
	_ map[string]any,
) (*engif.Result, error) {
	pr, ok := ent.(*pb.PullRequest)
	if !ok {
		return nil, fmt.Errorf("entity is not a pull request")
	}

	logger := zerolog.Ctx(ctx).With().
		Int32("pull-number", pr.Number).
		Str("repo-owner", pr.RepoOwner).
		Str("repo-name", pr.RepoName).
		Logger()

	allDiffs := make([]*pb.PrDependencies_ContextualDependency, 0)

	page := 0
	for {
		prFiles, resp, err := di.cli.ListFiles(ctx, pr.RepoOwner, pr.RepoName, int(pr.Number), prFilesPerPage, page)
		if err != nil {
			return nil, fmt.Errorf("error getting pull request files: %w", err)
		}

		for _, file := range prFiles {
			fileDiffs, err := di.ingestFile(file.GetFilename(), file.GetPatch(), file.GetRawURL(), logger)
			if err != nil {
				return nil, fmt.Errorf("error ingesting file %s: %w", file.GetFilename(), err)
			}
			allDiffs = append(allDiffs, fileDiffs...)
		}

		if resp.NextPage == 0 {
			break
		}

		page = resp.NextPage
	}

	return &engif.Result{
		Object: pb.PrDependencies{
			Pr:   pr,
			Deps: allDiffs,
		},
	}, nil
}

func (di *Diff) ingestFile(
	filename, patchContents, patchUrl string,
	logger zerolog.Logger,
) ([]*pb.PrDependencies_ContextualDependency, error) {
	parser := di.getParserForFile(filename, logger)
	if parser == nil {
		return nil, nil
	}

	depBatch, err := parser(patchContents)
	if err != nil {
		return nil, fmt.Errorf("error parsing file %s: %w", filename, err)
	}

	batchCtxDeps := make([]*pb.PrDependencies_ContextualDependency, 0, len(depBatch))
	for i := range depBatch {
		dep := depBatch[i]
		batchCtxDeps = append(batchCtxDeps, &pb.PrDependencies_ContextualDependency{
			Dep: dep,
			File: &pb.PrDependencies_ContextualDependency_FilePatch{
				Name:     filename,
				PatchUrl: patchUrl,
			},
		})
	}

	return batchCtxDeps, nil
}

func (di *Diff) getEcosystemForFile(filename string) DependencyEcosystem {
	lastComponent := filepath.Base(filename)

	for _, ecoMapping := range di.cfg.Ecosystems {
		if match, _ := filepath.Match(ecoMapping.Depfile, lastComponent); match {
			return DependencyEcosystem(ecoMapping.Name)
		}
	}
	return DepEcosystemNone
}

func (di *Diff) getParserForFile(filename string, logger zerolog.Logger) ecosystemParser {
	eco := di.getEcosystemForFile(filename)
	if eco == DepEcosystemNone {
		logger.Debug().
			Str("filename", filename).
			Msg("No ecosystem found, skipping")
		return nil
	}

	logger.Debug().
		Str("filename", filename).
		Str("package-ecosystem", string(eco)).
		Msg("matched ecosystem")

	return newEcosystemParser(eco)
}
