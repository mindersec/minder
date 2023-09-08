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

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"context"
	"fmt"
	"log"

	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

const (
	// VulncheckEvalType is the type of the vulncheck evaluator
	VulncheckEvalType = "vulncheck"
)

// Evaluator is the vulncheck evaluator
type Evaluator struct {
	cli ghclient.RestAPI
}

// NewVulncheckEvaluator creates a new vulncheck evaluator
func NewVulncheckEvaluator(_ *pb.RuleType_Definition_Eval_Vulncheck, cli ghclient.RestAPI) (*Evaluator, error) {
	return &Evaluator{
		cli: cli,
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

	for _, dep := range prdeps.Deps {
		ecoConfig := ruleConfig.getEcosystemConfig(dep.Dep.Ecosystem)
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

		pkgRepo, err := newRepository(ecoConfig)
		if err != nil {
			return fmt.Errorf("failed to create package repository: %w", err)
		}

		pkgReq, err := pkgRepo.NewRequest(ctx, dep.Dep)
		if err != nil {
			return fmt.Errorf("failed to create package request: %w", err)
		}

		patch, err := pkgRepo.SendRecvRequest(pkgReq)
		if err != nil {
			return fmt.Errorf("failed to send package request: %w", err)
		}

		switch ruleConfig.Action {
		case actionLog:
			log.Printf("vulncheck found vulnerabilities for %s", dep.Dep.Name)
		case actionRejectPr:
			reviewLoc, err := locateDepInPr(ctx, e.cli, dep)
			if err != nil {
				log.Printf("failed to locate dep in PR: %s", err)
				continue
			}

			err = requestChanges(ctx, e.cli, dep.File.Name, prdeps.Pr,
				reviewLoc, patch.IndentedString(reviewLoc.leadingWhitespace))
			if err != nil {
				log.Printf("failed to request changes on PR: %s", err)
				continue
			}
		case actionComment:
			log.Printf("not implemented")
		}
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
