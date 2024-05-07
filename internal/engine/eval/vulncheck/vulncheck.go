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
	"sort"
	"strings"

	"github.com/hashicorp/go-version"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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
func NewVulncheckEvaluator(ghcli provifv1.GitHub) (*Evaluator, error) {
	if ghcli == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	return &Evaluator{
		cli: ghcli,
	}, nil
}

// Eval implements the Evaluator interface.
func (e *Evaluator) Eval(ctx context.Context, pol map[string]any, res *engif.Result) error {
	vulnerablePackages, err := e.getVulnerableDependencies(ctx, pol, res)
	if err != nil {
		return err
	}

	if len(vulnerablePackages) > 0 {
		return evalerrors.NewErrEvaluationFailed(fmt.Sprintf("vulnerable packages: %s", strings.Join(vulnerablePackages, ",")))
	}

	return nil
}

// getVulnerableDependencies returns a slice containing vulnerable dependencies.
func (e *Evaluator) getVulnerableDependencies(ctx context.Context, pol map[string]any, res *engif.Result) ([]string, error) {
	var vulnerablePackages []string

	// TODO(jhrozek): Fix this!
	//nolint:govet
	prdeps, ok := res.Object.(*pb.PrDependencies)
	if !ok {
		return nil, fmt.Errorf("invalid object type for vulncheck evaluator")
	}

	if len(prdeps.Deps) == 0 {
		return nil, nil
	}

	ruleConfig, err := parseConfig(pol)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	prReplyHandler, err := newPrStatusHandler(ctx, ruleConfig.Action, prdeps.Pr, e.cli)
	if err != nil {
		return nil, fmt.Errorf("failed to create pr action: %w", err)
	}

	pkgRepoCache := newRepoCache()

	for _, dep := range prdeps.Deps {
		if dep.Dep == nil || dep.Dep.Version == "" {
			continue
		}

		vulnerable, err := e.checkVulnerabilities(ctx, dep, ruleConfig, pkgRepoCache, prReplyHandler)
		if err != nil {
			return nil, fmt.Errorf("failed to check vulnerabilities: %w", err)
		}

		if vulnerable {
			vulnerablePackages = append(vulnerablePackages, dep.Dep.Name)
		}
	}

	if err := prReplyHandler.submit(ctx); err != nil {
		return nil, fmt.Errorf("failed to submit pr action: %w", err)
	}

	return vulnerablePackages, nil
}

// getPatchedVersion returns a version that patches all known vulnerabilities. If no such version exists, it returns
// the version that patches the most vulnerabilities. If none of the vulnerabilities have patches, it returns the
// empty string.
func getPatchedVersion(vulns []Vulnerability) (fixedVersion string, latest bool, noFix bool) {
	var patches []*version.Version
	for _, vuln := range vulns {
		if vuln.Type == "SEMVER" {
			if vuln.Fixed != "" {
				newVersion, err := version.NewVersion(vuln.Fixed)
				if err == nil {
					patches = append(patches, newVersion)
				}
			}
		} else {
			// without semver we cannot tell which version is fixed, so return the latest
			if vuln.Fixed != "" {
				return "", true, false
			}
		}

	}
	if len(patches) == 0 {
		return "", false, true
	}
	sort.Sort(version.Collection(patches))
	return patches[len(patches)-1].String(), false, false
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

	response, err := db.SendRecvRequest(req, dep)
	if err != nil {
		return nil, fmt.Errorf("failed to send vulncheck request: %w", err)
	}

	return response, nil
}

// checkVulnerabilities checks whether a PR dependency contains any vulnerabilities.
func (e *Evaluator) checkVulnerabilities(
	ctx context.Context,
	dep *pb.PrDependencies_ContextualDependency,
	cfg *config,
	cache *repoCache,
	prHandler prStatusHandler,
) (bool, error) {
	ecoConfig := cfg.getEcosystemConfig(dep.Dep.Ecosystem)
	if ecoConfig == nil {
		fmt.Printf("Skipping dependency %s because ecosystem %s is not configured\n", dep.Dep.Name, dep.Dep.Ecosystem)
		return false, nil
	}

	vdb, err := e.getVulnDb(ecoConfig.DbType, ecoConfig.DbEndpoint)
	if err != nil {
		return false, fmt.Errorf("failed to get vulncheck db: %w", err)
	}

	response, err := e.queryVulnDb(ctx, vdb, dep.Dep, dep.Dep.Ecosystem)
	if err != nil {
		return false, fmt.Errorf("failed to query vulncheck db: %w", err)
	}

	if len(response.Vulns) == 0 {
		return false, nil
	}

	pkgRepo, err := cache.newRepository(ecoConfig)
	if err != nil {
		return false, fmt.Errorf("failed to create package repository: %w", err)
	}

	var patchFormatter patchLocatorFormatter
	if patched, latest, noFix := getPatchedVersion(response.Vulns); noFix {
		patchFormatter = pkgRepo.NoPatchAvailableFormatter(dep.Dep)
	} else if patchFormatter, err = pkgRepo.SendRecvRequest(ctx, dep.Dep, patched, latest); err != nil {
		return false, fmt.Errorf("failed to send package request: %w", err)
	}

	if err := prHandler.trackVulnerableDep(ctx, dep, response, patchFormatter); err != nil {
		return false, fmt.Errorf("failed to add package patch for further processing: %w", err)
	}

	return true, nil
}
