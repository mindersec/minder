// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	eoptions "github.com/mindersec/minder/internal/engine/options"
	pbinternal "github.com/mindersec/minder/internal/proto"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/flags"
)

const (
	// VulncheckEvalType is the type of the vulncheck evaluator
	VulncheckEvalType = "vulncheck"
)

// Evaluator is the vulncheck evaluator
type Evaluator struct {
	cli          GitHubRESTAndPRClient
	featureFlags flags.Interface
}

var _ eoptions.SupportsFlags = (*Evaluator)(nil)

// SetFlagsClient sets the `openfeature` client in the underlying
// `Evaluator` struct.
func (e *Evaluator) SetFlagsClient(client flags.Interface) error {
	e.featureFlags = client
	return nil
}

// NewVulncheckEvaluator creates a new vulncheck evaluator
func NewVulncheckEvaluator(
	ghcli GitHubRESTAndPRClient,
	opts ...interfaces.Option,
) (*Evaluator, error) {
	if ghcli == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	evaluator := &Evaluator{
		cli: ghcli,
	}

	for _, opt := range opts {
		if err := opt(evaluator); err != nil {
			return nil, err
		}
	}

	return evaluator, nil
}

// Eval implements the Evaluator interface.
func (e *Evaluator) Eval(
	ctx context.Context,
	pol map[string]any,
	_ protoreflect.ProtoMessage,
	res *interfaces.Ingested,
) (*interfaces.EvaluationResult, error) {
	vulnerablePackages, err := e.getVulnerableDependencies(ctx, pol, res)
	if err != nil {
		return nil, err
	}

	if len(vulnerablePackages) > 0 {
		return nil, evalerrors.NewDetailedErrEvaluationFailed(
			templates.VulncheckTemplate,
			map[string]any{"packages": vulnerablePackages},
			"vulnerable packages: %s",
			strings.Join(vulnerablePackages, ","),
		)
	}

	return &interfaces.EvaluationResult{}, nil
}

// getVulnerableDependencies returns a slice containing vulnerable dependencies.
// TODO: it would be nice if we could express this in rego over
// `input.ingested.deps[_].dep`, rather than building this in to core.
func (e *Evaluator) getVulnerableDependencies(
	ctx context.Context, pol map[string]any, res *interfaces.Ingested) ([]string, error) {
	var vulnerablePackages []string

	prdeps, ok := res.Object.(*pbinternal.PrDependencies)
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

func (*Evaluator) getVulnDb(dbType vulnDbType, endpoint string) (vulnDb, error) {
	switch dbType {
	case vulnDbTypeOsv:
		return newOsvDb(endpoint), nil
	default:
		return nil, fmt.Errorf("unsupported vulncheck db type: %s", dbType)
	}
}

func (*Evaluator) queryVulnDb(
	ctx context.Context,
	db vulnDb,
	dep *pbinternal.Dependency,
	ecosystem pbinternal.DepEcosystem,
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
	dep *pbinternal.PrDependencies_ContextualDependency,
	cfg *config,
	cache *repoCache,
	prHandler prStatusHandler,
) (bool, error) {
	ecoConfig := cfg.getEcosystemConfig(dep.Dep.Ecosystem)
	if ecoConfig == nil {
		zerolog.Ctx(ctx).Info().
			Str("ecosystem", string(dep.Dep.Ecosystem)).
			Str("dependency", dep.Dep.Name).
			Msg("Skipping dependency because ecosystem is not configured")
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
		patchFormatter = pkgRepo.PkgRegistryErrorFormatter(dep.Dep, err)
	}

	if err := prHandler.trackVulnerableDep(ctx, dep, response, patchFormatter); err != nil {
		return false, fmt.Errorf("failed to add package patch for further processing: %w", err)
	}

	return true, nil
}
