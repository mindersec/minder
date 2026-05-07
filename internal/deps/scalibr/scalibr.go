// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package scalibr implements a dependency extractor using the osv-scalibr
// library.
package scalibr

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"

	scalibr "github.com/google/osv-scalibr"
	scalibr_cfg "github.com/google/osv-scalibr/binary/proto/config_go_proto"
	scalibr_fs "github.com/google/osv-scalibr/fs"
	scalibr_plugin "github.com/google/osv-scalibr/plugin"
	"github.com/google/osv-scalibr/plugin/list"
	"github.com/google/uuid"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/rs/zerolog"
)

// Extractor is a dependency extractor based on osv-scalibr.
type Extractor struct {
}

// NewExtractor creates a new scalibr dependency extractor
func NewExtractor() *Extractor {
	return &Extractor{}
}

// ScanFilesystem takes
func (*Extractor) ScanFilesystem(ctx context.Context, iofs fs.FS) (*sbom.NodeList, error) {
	return scanFilesystem(ctx, iofs)
}

func scanFilesystem(ctx context.Context, iofs fs.FS) (*sbom.NodeList, error) {
	if iofs == nil {
		return nil, errors.New("unable to scan dependencies, no filesystem")
	}
	// have to down-cast here, because scalibr needs multiple io/fs types
	wrapped, ok := iofs.(scalibr_fs.FS)
	if !ok {
		return nil, errors.New("error converting filesystem to ReadDirFS")
	}

	desiredCaps := scalibr_plugin.Capabilities{
		OS:            scalibr_plugin.OSLinux,
		Network:       scalibr_plugin.NetworkOffline, // Don't fetch over the network, as we may be running in a trusted context.
		DirectFS:      false,
		RunningSystem: false,
	}

	// TODO: it's unfortunate that scalibr spills files to disk.  File an upstream bug?
	// NOTE: since we require NetworkOffline, we may not actually download anything...
	tmpDir, err := os.MkdirTemp("", "minder-scalibr-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary scalibr directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	cfg := scalibr_cfg.PluginConfig{
		MaxFileSizeBytes:  1024 * 1024,
		LocalRegistry:     tmpDir,
		DisableGoogleAuth: true,
	}

	scalibrFs := scalibr_fs.ScanRoot{FS: wrapped}
	plugins, err := list.FromCapabilities(&desiredCaps, &cfg)
	if err != nil {
		return nil, err
	}
	// unknownbinariesextr uses file extension to determine "binary-ness", and triggers on e.g. .py files
	skipPlugins := []string{"ffa/unknownbinariesextr"}
	plugins = slices.DeleteFunc(plugins, func(p scalibr_plugin.Plugin) bool {
		return slices.Contains(skipPlugins, p.Name())
	})
	scanConfig := scalibr.ScanConfig{
		ScanRoots:    []*scalibr_fs.ScanRoot{&scalibrFs},
		Plugins:      plugins,
		Capabilities: &desiredCaps,
	}

	scanner := scalibr.New()
	scanResults := scanner.Scan(ctx, &scanConfig)

	if scanResults == nil || scanResults.Status == nil {
		return nil, fmt.Errorf("error scanning files: no results")
	}
	switch scanResults.Status.Status {
	case scalibr_plugin.ScanStatusSucceeded:
		// success, continue
	case scalibr_plugin.ScanStatusPartiallySucceeded:
		// Scalibr runs a lot of plugins and aggregates the result.  Some of these are picky, and
		// fail for random reasons.  Accept partial success, but log the failing plugins.
		known_bad := []string{
			"endoflife/linuxdistro", // https://github.com/google/osv-scalibr/pull/2068
			"rust/cargoauditable",   // https://github.com/go-git/go-billy/pull/208
		}
		for _, ps := range scanResults.PluginStatus {
			if ps.Status.Status != scalibr_plugin.ScanStatusSucceeded {
				if !slices.Contains(known_bad, ps.Name) {
					zerolog.Ctx(ctx).Warn().Str("plugin", ps.Name).Str("status", ps.Status.FailureReason).
						Msg("Scalibr plugin failed")
				}
			}
		}
	case scalibr_plugin.ScanStatusUnspecified, scalibr_plugin.ScanStatusFailed:
		fallthrough
	default:
		return nil, fmt.Errorf("error scanning files: %s", scanResults.Status)
	}

	res := sbom.NewNodeList()
	for _, inv := range scanResults.Inventory.Packages {
		// TODO: use repo and commit from inv.SourceCode
		node := &sbom.Node{
			Type:    sbom.Node_PACKAGE,
			Id:      uuid.New().String(),
			Name:    inv.Name,
			Version: inv.Version,
			Identifiers: map[int32]string{
				int32(sbom.SoftwareIdentifierType_PURL): inv.PURL().String(),
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
