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
	"slices"

	scalibr "github.com/google/osv-scalibr"
	"github.com/google/osv-scalibr/extractor/filesystem/language/golang/gobinary"
	scalibr_fs "github.com/google/osv-scalibr/fs"
	scalibr_plugin "github.com/google/osv-scalibr/plugin"
	"github.com/google/osv-scalibr/plugin/list"
	"github.com/google/uuid"
	"github.com/protobom/protobom/pkg/sbom"
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

	scalibrFs := scalibr_fs.ScanRoot{FS: wrapped}
	extractors := list.FromCapabilities(&desiredCaps)
	// Don't run the go binary extractor; it sometimes panics on certain files.
	extractors = slices.DeleteFunc(extractors, func(e scalibr_plugin.Plugin) bool {
		_, ok := e.(*gobinary.Extractor)
		return ok
	})
	scanConfig := scalibr.ScanConfig{
		ScanRoots: []*scalibr_fs.ScanRoot{&scalibrFs},
		// All includes Ruby, Dotnet which we're not ready to test yet, so use the more limited Default set.
		Plugins:      extractors,
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
