// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package deps abstracts a dependency extractor
package deps

import (
	"context"
	"io/fs"

	"github.com/protobom/protobom/pkg/sbom"

	"github.com/mindersec/minder/internal/deps/scalibr"
)

var _ Extractor = (*scalibr.Extractor)(nil)

// Extractor is the object that groups the dependency extractor. It shields the
// implementations that Minder uses behinf a common interface to extract depencies
// from filesystems.
type Extractor interface {
	ScanFilesystem(context.Context, fs.FS) (*sbom.NodeList, error)
}
