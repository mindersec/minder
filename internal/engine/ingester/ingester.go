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
// Package rule provides the CLI subcommand for managing rules

// Package ingester provides necessary interfaces and implementations for ingesting
// data for rules.
package ingester

import (
	"fmt"

	"github.com/stacklok/mediator/internal/engine/ingester/artifact"
	"github.com/stacklok/mediator/internal/engine/ingester/builtin"
	"github.com/stacklok/mediator/internal/engine/ingester/diff"
	"github.com/stacklok/mediator/internal/engine/ingester/git"
	"github.com/stacklok/mediator/internal/engine/ingester/rest"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// test that the ingester implementations implements the interface
// this would be probably nicer in the implementation file, but that would cause an import loop
var _ engif.Ingester = (*artifact.Ingest)(nil)
var _ engif.Ingester = (*builtin.BuiltinRuleDataIngest)(nil)
var _ engif.Ingester = (*rest.Ingestor)(nil)

// NewRuleDataIngest creates a new rule data ingest based no the given rule
// type definition.
func NewRuleDataIngest(rt *pb.RuleType, cli ghclient.RestAPI, access_token string) (engif.Ingester, error) {
	ing := rt.Def.GetIngest()
	switch rt.Def.Ingest.Type {
	case rest.RestRuleDataIngestType:
		if rt.Def.Ingest.GetRest() == nil {
			return nil, fmt.Errorf("rule type engine missing rest configuration")
		}

		return rest.NewRestRuleDataIngest(ing.GetRest(), cli)
	case builtin.BuiltinRuleDataIngestType:
		if rt.Def.Ingest.GetBuiltin() == nil {
			return nil, fmt.Errorf("rule type engine missing internal configuration")
		}
		return builtin.NewBuiltinRuleDataIngest(ing.GetBuiltin(), access_token)

	case artifact.ArtifactRuleDataIngestType:
		if rt.Def.Ingest.GetArtifact() == nil {
			return nil, fmt.Errorf("rule type engine missing artifact configuration")
		}
		return artifact.NewArtifactDataIngest(ing.GetArtifact())

	case git.GitRuleDataIngestType:
		return git.NewGitIngester(ing.GetGit(), access_token), nil
	case diff.DiffRuleDataIngestType:
		return diff.NewDiffIngester(ing.GetDiff(), cli), nil
	default:
		return nil, fmt.Errorf("unsupported rule type engine: %s", rt.Def.Ingest.Type)
	}
}
