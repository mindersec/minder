// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
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
	"log"

	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	ghclient "github.com/stacklok/mediator/internal/providers/github"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const (
	// DiffRuleDataIngestType is the type of the diff rule data ingest engine
	DiffRuleDataIngestType = "diff"
)

// Diff is the diff rule data ingest engine
type Diff struct {
	cli ghclient.RestAPI
	cfg *pb.DiffType
}

// NewDiffIngester creates a new diff ingester
func NewDiffIngester(
	cfg *pb.DiffType,
	cli ghclient.RestAPI,
) *Diff {
	if cfg == nil {
		cfg = &pb.DiffType{}
	}
	return &Diff{
		cfg: cfg,
		cli: cli,
	}
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

	// TODO(jakub): support pagination
	prFiles, err := di.cli.ListFiles(ctx, pr.RepoOwner, pr.RepoName, int(pr.Number), 1, 100)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request files: %w", err)
	}

	allDiffs := make([]*pb.PrDependencies_ContextualDependency, 0)

	for _, file := range prFiles {
		eco := di.getEcosystemForFile(*file.Filename)
		if eco == DepEcosystemNone {
			log.Printf("no ecosystem found for file %s", *file.Filename)
			continue
		}

		parser := newEcosystemParser(eco)
		if parser == nil {
			return nil, fmt.Errorf("no parser found for ecosystem %s", eco)
		}

		depBatch, err := parser(*file.Patch)
		if err != nil {
			return nil, fmt.Errorf("error parsing file %s: %w", *file.Filename, err)
		}

		for i := range depBatch {
			dep := depBatch[i]
			allDiffs = append(allDiffs, &pb.PrDependencies_ContextualDependency{
				Dep: dep,
				File: &pb.FilePatch{
					Name:     file.GetFilename(),
					PatchUrl: file.GetRawURL(),
				},
			})
		}
	}

	return &engif.Result{
		Object: pb.PrDependencies{
			Pr:   pr,
			Deps: allDiffs,
		},
	}, nil
}

func (di *Diff) getEcosystemForFile(filename string) DependencyEcosystem {
	for _, ecoMapping := range di.cfg.Ecosystems {
		if filename == ecoMapping.Depfile {
			return DependencyEcosystem(ecoMapping.Name)
		}
	}
	return DepEcosystemNone
}
