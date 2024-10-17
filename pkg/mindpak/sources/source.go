// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package sources contains logic for loading a bundle from a source of bundles
package sources

import (
	"errors"
	"fmt"

	"github.com/mindersec/minder/pkg/mindpak"
	"github.com/mindersec/minder/pkg/mindpak/reader"
)

// BundleSource contains methods for retrieving bundles. Implementations may
// load the Bundle from disk, an OCI registry, or any other place where bundles
// may be present. Bundles are returned as instances of the BundleReader
// interface.
type BundleSource interface {
	// GetBundle fetches a bundle from the source by namespace and name
	// ErrBundleNotFound is returned if the bundle cannot be found
	GetBundle(id mindpak.BundleID) (reader.BundleReader, error)
	// ListBundles lists all bundles known to this source
	ListBundles() ([]mindpak.BundleID, error)
}

var (
	// ErrBundleNotFound is returned when the specified bundle is not found
	ErrBundleNotFound = errors.New("bundle not found")
)

// NewSourceFromTarGZ creates a singleBundleSource from a .tar.gz file
func NewSourceFromTarGZ(path string) (BundleSource, error) {
	bundle, err := mindpak.NewBundleFromTarGZ(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load bundle from %s: %w", path, err)
	}
	if err := bundle.Verify(); err != nil {
		return nil, fmt.Errorf("bundle failed verification: %w", err)
	}
	bundleReader := reader.NewBundleReader(bundle)
	return &singleBundleSource{bundle: bundleReader}, nil
}

// singleBundleSource is a trivial implementation of BundleSource for a single
// bundle
type singleBundleSource struct {
	bundle reader.BundleReader
}

func (s *singleBundleSource) GetBundle(id mindpak.BundleID) (reader.BundleReader, error) {
	metadata := s.bundle.GetMetadata()
	if id.Namespace == metadata.Namespace && id.Name == metadata.Name {
		return s.bundle, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrBundleNotFound, id)
}

func (s *singleBundleSource) ListBundles() ([]mindpak.BundleID, error) {
	return []mindpak.BundleID{
		mindpak.ID(s.bundle.GetMetadata().Namespace, s.bundle.GetMetadata().Name),
	}, nil
}
