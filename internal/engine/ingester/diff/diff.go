// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package diff provides the diff rule data ingest engine
package diff

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/iofs"
	scalibr "github.com/google/osv-scalibr"
	"github.com/google/osv-scalibr/extractor"
	scalibr_fs "github.com/google/osv-scalibr/fs"
	scalibr_plugin "github.com/google/osv-scalibr/plugin"
	"github.com/google/osv-scalibr/plugin/list"
	"github.com/google/osv-scalibr/purl"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	pbinternal "github.com/mindersec/minder/internal/proto"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/entities/v1/checkpoints"
)

const (
	// DiffRuleDataIngestType is the type of the diff rule data ingest engine
	DiffRuleDataIngestType = "diff"
	prFilesPerPage         = 30
	wildcard               = "*"
)

// Diff is the diff rule data ingest engine
type Diff struct {
	cli interfaces.GitHubListAndClone
	cfg *pb.DiffType
}

// NewDiffIngester creates a new diff ingester
func NewDiffIngester(
	cfg *pb.DiffType,
	cli interfaces.GitHubListAndClone,
) (*Diff, error) {
	if cfg == nil {
		cfg = &pb.DiffType{}
	}

	if cli == nil {
		return nil, fmt.Errorf("provider is nil")
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

// Ingest ingests a diff from a pull request in accordance with its type
func (di *Diff) Ingest(
	ctx context.Context,
	ent protoreflect.ProtoMessage,
	_ map[string]any,
) (*interfaces.Ingested, error) {
	pr, ok := ent.(*pbinternal.PullRequest)
	if !ok {
		return nil, fmt.Errorf("entity is not a pull request")
	}

	// The GitHub Go API takes an int32, but our proto stores an int64; make sure we don't overflow
	if pr.Number > math.MaxInt {
		return nil, fmt.Errorf("pr number is too large")
	}
	prNumber := int(pr.Number)

	switch di.cfg.GetType() {
	case "", pb.DiffTypeDep:
		return di.getDepTypeDiff(ctx, prNumber, pr)

	case pb.DiffTypeNewDeps:
		// TODO: once we've tested some, convert DiffTypeDep to use this algorithm.
		return di.getScalibrTypeDiff(ctx, prNumber, pr)

	case pb.DiffTypeFull:
		return di.getFullTypeDiff(ctx, prNumber, pr)

	default:
		return nil, fmt.Errorf("unknown diff type")
	}
}

func (di *Diff) getDepTypeDiff(ctx context.Context, prNumber int, pr *pbinternal.PullRequest) (*interfaces.Ingested, error) {
	deps := pbinternal.PrDependencies{Pr: pr}
	page := 0

	for {
		prFiles, resp, err := di.cli.ListFiles(ctx, pr.RepoOwner, pr.RepoName, prNumber, prFilesPerPage, page)
		if err != nil {
			return nil, fmt.Errorf("error getting pull request files: %w", err)
		}

		for _, file := range prFiles {
			fileDiffs, err := di.ingestFileForDepDiff(file.GetFilename(), file.GetPatch(), file.GetRawURL(), *zerolog.Ctx(ctx))
			if err != nil {
				return nil, fmt.Errorf("error ingesting file %s: %w", file.GetFilename(), err)
			}
			deps.Deps = append(deps.Deps, fileDiffs...)
		}

		if resp.NextPage == 0 {
			break
		}

		page = resp.NextPage
	}

	return &interfaces.Ingested{Object: &deps, Checkpoint: checkpoints.NewCheckpointV1Now()}, nil
}

func (di *Diff) getFullTypeDiff(ctx context.Context, prNumber int, pr *pbinternal.PullRequest) (*interfaces.Ingested, error) {
	diff := &pbinternal.PrContents{Pr: pr}
	page := 0

	for {
		prFiles, resp, err := di.cli.ListFiles(ctx, pr.RepoOwner, pr.RepoName, prNumber, prFilesPerPage, page)
		if err != nil {
			return nil, fmt.Errorf("error getting pull request files: %w", err)
		}

		for _, file := range prFiles {
			fileDiffs, err := ingestFileForFullDiff(file.GetFilename(), file.GetPatch(), file.GetRawURL())
			if err != nil {
				return nil, fmt.Errorf("error ingesting file %s: %w", file.GetFilename(), err)
			}
			diff.Files = append(diff.Files, fileDiffs)
		}

		if resp.NextPage == 0 {
			break
		}

		page = resp.NextPage
	}

	return &interfaces.Ingested{Object: diff, Checkpoint: checkpoints.NewCheckpointV1Now()}, nil
}

func (di *Diff) ingestFileForDepDiff(
	filename, patchContents, patchUrl string,
	logger zerolog.Logger,
) ([]*pbinternal.PrDependencies_ContextualDependency, error) {
	parser := di.getParserForFile(filename, logger)
	if parser == nil {
		return nil, nil
	}

	depBatch, err := parser(patchContents)
	if err != nil {
		return nil, fmt.Errorf("error parsing file %s: %w", filename, err)
	}

	batchCtxDeps := make([]*pbinternal.PrDependencies_ContextualDependency, 0, len(depBatch))
	for i := range depBatch {
		dep := depBatch[i]
		batchCtxDeps = append(batchCtxDeps, &pbinternal.PrDependencies_ContextualDependency{
			Dep: dep,
			File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
				Name:     filename,
				PatchUrl: patchUrl,
			},
		})
	}

	return batchCtxDeps, nil
}

func (di *Diff) getScalibrTypeDiff(ctx context.Context, _ int, pr *pbinternal.PullRequest) (*interfaces.Ingested, error) {
	deps := pbinternal.PrDependencies{Pr: pr}

	// TODO: we should be able to just fetch the additional commits between base and target.
	// Our current Git abstraction isn't quite powerful enough, so we do two shallow clones.

	baseInventory, err := di.scalibrInventory(ctx, pr.BaseCloneUrl, pr.BaseRef)
	if err != nil {
		return nil, fmt.Errorf("failed to clone base from %s at %q: %w", pr.BaseCloneUrl, pr.BaseRef, err)
	}
	newInventory, err := di.scalibrInventory(ctx, pr.TargetCloneUrl, pr.TargetRef)
	if err != nil {
		return nil, fmt.Errorf("failed to clone fork from %s at %q: %w", pr.TargetCloneUrl, pr.TargetRef, err)
	}

	newDeps := setDifference(baseInventory, newInventory, inventorySorter)

	deps.Deps = make([]*pbinternal.PrDependencies_ContextualDependency, 0, len(newDeps))
	for _, inventory := range newDeps {
		for _, filename := range inventory.Locations {
			deps.Deps = append(deps.Deps, &pbinternal.PrDependencies_ContextualDependency{
				Dep: &pbinternal.Dependency{
					Ecosystem: inventoryToEcosystem(inventory),
					Name:      inventory.Name,
					Version:   inventory.Version,
				},
				File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
					Name:     filename,
					PatchUrl: "", // TODO: do we need this?
				},
			})
		}
	}

	return &interfaces.Ingested{Object: &deps, Checkpoint: checkpoints.NewCheckpointV1Now()}, nil
}

func inventorySorter(a *extractor.Package, b *extractor.Package) int {
	// If we compare by name and version first, we can avoid serializing Locations to strings
	res := cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Version, b.Version))
	if res != 0 {
		return res
	}
	// TODO: Locations should probably be sorted, but scalibr is going to export a compare function.
	aLoc := fmt.Sprintf("%v", a.Locations)
	bLoc := fmt.Sprintf("%v", b.Locations)
	return cmp.Compare(aLoc, bLoc)
}

func (di *Diff) scalibrInventory(ctx context.Context, repoURL string, ref string) ([]*extractor.Package, error) {
	clone, err := di.cli.Clone(ctx, repoURL, ref)
	if err != nil {
		return nil, err
	}

	tree, err := clone.Worktree()
	if err != nil {
		return nil, err
	}
	return scanFs(ctx, tree.Filesystem, map[string]string{})
}

func scanFs(ctx context.Context, memFS billy.Filesystem, _ map[string]string) ([]*extractor.Package, error) {
	// have to down-cast here, because scalibr needs multiple io/fs types
	wrapped, ok := iofs.New(memFS).(scalibr_fs.FS)
	if !ok {
		return nil, fmt.Errorf("error converting filesystem to ReadDirFS")
	}

	desiredCaps := scalibr_plugin.Capabilities{
		OS:            scalibr_plugin.OSLinux,
		Network:       scalibr_plugin.NetworkOffline,
		DirectFS:      false,
		RunningSystem: false,
	}

	scalibrFs := scalibr_fs.ScanRoot{FS: wrapped}
	scanConfig := scalibr.ScanConfig{
		ScanRoots: []*scalibr_fs.ScanRoot{&scalibrFs},
		// All includes Ruby, Dotnet which we're not ready to test yet, so use the more limited Default set.
		Plugins:      list.FromCapabilities(&desiredCaps),
		Capabilities: &desiredCaps,
	}

	scanner := scalibr.New()
	scanResults := scanner.Scan(ctx, &scanConfig)

	if scanResults == nil || scanResults.Status == nil {
		return nil, fmt.Errorf("error scanning files: no results")
	}
	if scanResults.Status.Status != scalibr_plugin.ScanStatusSucceeded {
		return nil, fmt.Errorf("error scanning files: %s", scanResults.Status)
	}

	return scanResults.Inventory.Packages, nil
}

func inventoryToEcosystem(inventory *extractor.Package) pbinternal.DepEcosystem {
	if inventory == nil {
		zerolog.Ctx(context.Background()).Warn().Msg("nil ecosystem scanning diffs")
		return pbinternal.DepEcosystem_DEP_ECOSYSTEM_UNSPECIFIED
	}

	package_url := inventory.PURL()

	// Sometimes Scalibr uses the string "PyPI" instead of "pypi" when reporting the ecosystem.
	switch package_url.Type {
	// N.B. using an enum here abitrarily restricts our ability to add new
	// ecosystems without a core minder change.  Switching to strings ala
	// purl might be an improvement.
	case purl.TypePyPi:
		return pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI
	case purl.TypeNPM:
		return pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM
	case purl.TypeGolang:
		return pbinternal.DepEcosystem_DEP_ECOSYSTEM_GO
	default:
		return pbinternal.DepEcosystem_DEP_ECOSYSTEM_UNSPECIFIED
	}
}

// ingestFileForFullDiff processes a given file's patch from a pull request.
// It scans through the patch line by line, identifying the changes made.
// If it's a hunk header, it extracts the starting line number. If it's an addition, it records the line content and its number.
// The function also increments the line number for context lines (lines that provide context but haven't been modified).
func ingestFileForFullDiff(filename, patch, patchUrl string) (*pbinternal.PrContents_File, error) {
	var result []*pbinternal.PrContents_File_Line

	scanner := bufio.NewScanner(strings.NewReader(patch))
	regex := regexp.MustCompile(`@@ -\d+,\d+ \+(\d+),\d+ @@`)

	var currentLineNumber int64
	var err error
	for scanner.Scan() {
		line := scanner.Text()

		if matches := regex.FindStringSubmatch(line); matches != nil {
			currentLineNumber, err = strconv.ParseInt(matches[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("error parsing line number from the hunk header: %w", err)
			}
		} else if strings.HasPrefix(line, "+") {
			result = append(result, &pbinternal.PrContents_File_Line{
				Content: line[1:],
				// see the use of strconv.ParseInt above: this is a safe downcast
				// nolint: gosec
				LineNumber: int32(currentLineNumber),
			})

			currentLineNumber++
		} else if !strings.HasPrefix(line, "-") {
			currentLineNumber++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading patch: %w", err)
	}

	return &pbinternal.PrContents_File{
		Name:         filename,
		FilePatchUrl: patchUrl,
		PatchLines:   result,
	}, nil
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

// Computes the set of elements in updated which are not in base.
// Note: this function may permute (sort) the order of elements in base and updated.
func setDifference[Slice ~[]E, E any](base Slice, updated Slice, sorter func(a, b E) int) Slice {

	slices.SortFunc(base, sorter)
	slices.SortFunc(updated, sorter)

	baseIdx, newIdx := 0, 0
	ret := make(Slice, 0)
	for baseIdx < len(base) && newIdx < len(updated) {
		cmpResult := sorter(base[baseIdx], updated[newIdx])
		if cmpResult < 0 {
			baseIdx++
		} else if cmpResult > 0 {
			ret = append(ret, updated[newIdx])
			newIdx++
		} else {
			baseIdx++
			newIdx++
		}
	}
	if newIdx < len(updated) {
		ret = append(ret, updated[newIdx:]...)
	}

	// TODO: add metric for number of deps scanned vs total deps

	return ret
}
