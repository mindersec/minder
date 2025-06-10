// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ingester provides necessary interfaces and implementations for ingesting
// data for rules.
package ingester

import (
	"errors"
	"fmt"

	"github.com/mindersec/minder/internal/engine/ingester/artifact"
	"github.com/mindersec/minder/internal/engine/ingester/builtin"
	"github.com/mindersec/minder/internal/engine/ingester/deps"
	"github.com/mindersec/minder/internal/engine/ingester/diff"
	"github.com/mindersec/minder/internal/engine/ingester/git"
	"github.com/mindersec/minder/internal/engine/ingester/rest"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// test that the ingester implementations implements the interface
// this would be probably nicer in the implementation file, but that would cause an import loop
var _ interfaces.Ingester = (*artifact.Ingest)(nil)
var _ interfaces.Ingester = (*builtin.RuleDataIngest)(nil)
var _ interfaces.Ingester = (*rest.Ingestor)(nil)
var _ interfaces.Ingester = (*git.Git)(nil)
var _ interfaces.Ingester = (*diff.Diff)(nil)
var _ interfaces.Ingester = (*deps.Deps)(nil)

// NewRuleDataIngest creates a new rule data ingest based no the given rule
// type definition.
func NewRuleDataIngest(rt *pb.RuleType, provider interfaces.Provider) (interfaces.Ingester, error) {
	ing := rt.Def.GetIngest()

	switch ing.GetType() {
	case rest.RestRuleDataIngestType:
		if rt.Def.Ingest.GetRest() == nil {
			return nil, fmt.Errorf("rule type engine missing rest configuration")
		}
		client, err := interfaces.As[interfaces.RESTProvider](provider)
		if err != nil {
			return nil, errors.New("provider does not implement rest trait")
		}

		return rest.NewRestRuleDataIngest(ing.GetRest(), client)
	case builtin.BuiltinRuleDataIngestType:
		if rt.Def.Ingest.GetBuiltin() == nil {
			return nil, fmt.Errorf("rule type engine missing internal configuration")
		}
		return builtin.NewRuleDataIngest(ing.GetBuiltin())

	case artifact.ArtifactRuleDataIngestType:
		if rt.Def.Ingest.GetArtifact() == nil {
			return nil, fmt.Errorf("rule type engine missing artifact configuration")
		}
		return artifact.NewArtifactDataIngest(provider)

	case git.GitRuleDataIngestType:
		client, err := interfaces.As[interfaces.GitProvider](provider)
		if err != nil {
			return nil, errors.New("provider does not implement git trait")
		}
		return git.NewGitIngester(ing.GetGit(), client)
	case diff.DiffRuleDataIngestType:
		client, err := interfaces.As[interfaces.GitHubListAndClone](provider)
		if err != nil {
			return nil, errors.New("provider does not implement github trait")
		}
		return diff.NewDiffIngester(ing.GetDiff(), client)
	case deps.DepsRuleDataIngestType:
		client, err := interfaces.As[interfaces.GitProvider](provider)
		if err != nil {
			return nil, errors.New("provider does not implement git trait")
		}
		return deps.NewDepsIngester(ing.GetDeps(), client)
	default:
		return nil, fmt.Errorf("unsupported rule type engine: %s", rt.Def.Ingest.Type)
	}
}
