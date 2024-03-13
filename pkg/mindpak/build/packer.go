//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package build implements tools and function to build mindpaks. The main
// builder is build.Packer that writes the bundles to archives.
package build

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/pkg/mindpak"
)

// Packer handles writing the bundles to archives on disk.
type Packer struct{}

// NewPacker returns a new packer object with the default options
func NewPacker() *Packer {
	return &Packer{}
}

// InitOptions are used when initializing a new bundle directory.
type InitOptions struct {
	*mindpak.Metadata
	Path string
}

// Validate checks the initializer options
func (opts *InitOptions) Validate() error {
	var errs = []error{}
	if opts.Name == "" {
		errs = append(errs, fmt.Errorf("name is required to initialize a mindpack"))
	} else if !mindpak.ValidNameRegex.MatchString(opts.Name) {
		errs = append(errs, fmt.Errorf("%q is not a valid mindpack name", opts.Name))
	}

	if opts.Namespace != "" && !mindpak.ValidNameRegex.MatchString(opts.Namespace) {
		errs = append(errs, fmt.Errorf("%q is not valida namespace", opts.Namespace))
	}

	// FIXME(puerco): Check semver

	// Check path
	sdata, err := os.Stat(opts.Path)
	if err != nil {
		errs = append(errs, fmt.Errorf("opening path: %w", err))
	} else {
		if !sdata.IsDir() {
			errs = append(errs, fmt.Errorf("path is not a directory"))
		}
	}

	return errors.Join(errs...)
}

// Init creates a new bundle manifest in a directory with minder data in the
// expected structure.
func (_ *Packer) Init(opts *InitOptions) error {
	if opts.Metadata.Name == "" {
		return fmt.Errorf("unable to initialize new bundle, no name defined")
	}

	bundle, err := mindpak.NewBundleFromDirectory(opts.Path)
	if err != nil {
		return fmt.Errorf("reading source data: %w", err)
	}

	bundle.Metadata = opts.Metadata

	if err := bundle.UpdateManifest(); err != nil {
		return fmt.Errorf("updating new bundle manifest: %w", err)
	}

	bundle.Metadata.Date = timestamppb.Now()

	f, err := os.Create(filepath.Join(opts.Path, mindpak.ManifestFileName))
	if err != nil {
		return fmt.Errorf("opening manifest file: %w", err)
	}

	if err := bundle.Manifest.Write(f); err != nil {
		return fmt.Errorf("writing manifest data: %w", err)
	}

	return nil
}

// WriteToFile writes the bundle to a file on disk.
func (p *Packer) WriteToFile(bundle *mindpak.Bundle, path string) error {
	path = filepath.Clean(path)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	if err := p.Write(bundle, f); err != nil {
		return fmt.Errorf("writing bundle to file: %w", err)
	}
	return nil
}

// Write writes a bundle archive to writer w
func (_ *Packer) Write(bundle *mindpak.Bundle, w io.Writer) error {
	tarWriter := tar.NewWriter(w)
	defer tarWriter.Close()

	if bundle.Source == nil {
		return fmt.Errorf("unable to pack bundle, data source not defined")
	}

	err := fs.WalkDir(bundle.Source, ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("reading %q: %w", path, err)
		}

		stat, err := fs.Stat(bundle.Source, path)
		if err != nil {
			return fmt.Errorf("reading file info: %w", err)
		}
		if stat.IsDir() {
			return nil
		}

		f, err := bundle.Source.Open(path)
		if err != nil {
			return fmt.Errorf("opening %q", path)
		}
		defer f.Close()

		header := &tar.Header{
			Name:    path,
			Size:    stat.Size(),
			Mode:    int64(stat.Mode()),
			ModTime: stat.ModTime(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("writing header for %q: %w", path, err)
		}

		if _, err := io.Copy(tarWriter, f); err != nil {
			return fmt.Errorf("writing data from %q to archive: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("walking bundle data source: %w", err)
	}

	return nil
}
