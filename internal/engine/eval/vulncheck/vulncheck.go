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

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"context"
	"fmt"

	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
	provifv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

const (
	// VulncheckEvalType is the type of the vulncheck evaluator
	VulncheckEvalType = "vulncheck"
)

// Evaluator is the vulncheck evaluator
type Evaluator struct {
	cli provifv1.GitHub
}

// NewVulncheckEvaluator creates a new vulncheck evaluator
func NewVulncheckEvaluator(_ *pb.RuleType_Definition_Eval_Vulncheck, pbuild *providers.ProviderBuilder) (*Evaluator, error) {
	if pbuild == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	ghcli, err := pbuild.GetGitHub(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get github client: %w", err)
	}

	return &Evaluator{
		cli: ghcli,
	}, nil
}

// Eval implements the Evaluator interface.
func (e *Evaluator) Eval(ctx context.Context, pol map[string]any, res *engif.Result) error {
	var evalErr error

	// TODO(jhrozek): Fix this!
	//nolint:govet
	prdeps, ok := res.Object.(pb.PrDependencies)
	if !ok {
		return fmt.Errorf("invalid object type for vulncheck evaluator")
	}

	ruleConfig, err := parseConfig(pol)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	prReplyHandler, err := newPrStatusHandler(ctx, ruleConfig.Action, prdeps.Pr, e.cli)
	if err != nil {
		return fmt.Errorf("failed to create pr action: %w", err)
	}

	pkgRepoCache := newRepoCache()

	for _, dep := range prdeps.Deps {
		if dep.Dep == nil || dep.Dep.Version == "" {
			continue
		}

		ecoConfig := ruleConfig.getEcosystemConfig(dep.Dep.Ecosystem)
		if ecoConfig == nil {
			fmt.Printf("Skipping dependency %s because ecosystem %s is not configured\n", dep.Dep.Name, dep.Dep.Ecosystem)
			continue
		}

		vdb, err := e.getVulnDb(ecoConfig.DbType, ecoConfig.DbEndpoint)
		if err != nil {
			return fmt.Errorf("failed to get vulncheck db: %w", err)
		}

		response, err := e.queryVulnDb(ctx, vdb, dep.Dep, dep.Dep.Ecosystem)
		if err != nil {
			return fmt.Errorf("failed to query vulncheck db: %w", err)
		}

		if len(response.Vulns) == 0 {
			continue
		}

		// TODO(jhrozek): this should be a list of vulnerabilities
		evalErr = fmt.Errorf("vulnerabilities found for %s", dep.Dep.Name)

		pkgRepo, err := pkgRepoCache.newRepository(ecoConfig)
		if err != nil {
			return fmt.Errorf("failed to create package repository: %w", err)
		}

		patch, err := pkgRepo.SendRecvRequest(ctx, dep.Dep)
		if err != nil {
			return fmt.Errorf("failed to send package request: %w", err)
		}

		if err := prReplyHandler.trackVulnerableDep(ctx, dep, response, patch); err != nil {
			return fmt.Errorf("failed to add package patch for further processing: %w", err)
		}
	}

	if err := prReplyHandler.submit(ctx); err != nil {
		return fmt.Errorf("failed to submit pr action: %w", err)
	}

	return evalErr
}

func (_ *Evaluator) getVulnDb(dbType vulnDbType, endpoint string) (vulnDb, error) {
	switch dbType {
	case vulnDbTypeOsv:
		return newOsvDb(endpoint), nil
	default:
		return nil, fmt.Errorf("unsupported vulncheck db type: %s", dbType)
	}
}

func (_ *Evaluator) queryVulnDb(
	ctx context.Context,
	db vulnDb,
	dep *pb.Dependency,
	ecosystem pb.DepEcosystem,
) (*VulnerabilityResponse, error) {
	req, err := db.NewQuery(ctx, dep, ecosystem)
	if err != nil {
		return nil, fmt.Errorf("failed to create vulncheck request: %w", err)
	}

	response, err := db.SendRecvRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send vulncheck request: %w", err)
	}

	return response, nil
}
