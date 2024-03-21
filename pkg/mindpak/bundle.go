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

// Package mindpak abstracts to bundle profiles and rule types together in
// an artifact that allows for easy distribution and upgrade.
package mindpak

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
)

var (
	// ValidNameRegex is used to check a bundle name or namespace
	ValidNameRegex = regexp.MustCompile(`^[a-zA-Z0-9](?:[-_a-zA-Z0-9]{0,61}[a-zA-Z0-9])?$`)
)

// Bundle abstracts the bundle data.
//
// The bundle has a manifest. The manifest is composed by reading the
// Source filesystem and categorizing its entries.
type Bundle struct {
	Manifest *Manifest
	Metadata *Metadata
	Files    *Files
	Source   fs.StatFS
}

// NewBundleFromDirectory reads a directory from a directory and returns
// a bundle loaded with its contents. The bundle will have its Source filesystem
// bound to the directory via an os.DirFS.
func NewBundleFromDirectory(path string) (*Bundle, error) {
	i, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("opening bundle directory: %w", err)
	}
	if !i.IsDir() {
		return nil, fmt.Errorf("specified path is not a directory")
	}

	bundle := &Bundle{
		Source: os.DirFS(path).(fs.StatFS),
	}
	if err := bundle.ReadSource(); err != nil {
		return nil, fmt.Errorf("reading bundle data from %q: %w", path, err)
	}

	return bundle, nil
}

// NewBundleFromTarGZ loads a bundle from a .tar.gz file containing the bundle
// structure. Note that this implementation loads the entire contents of the
// bundle into memory.
// This is tested by the test in the `internal/marketplace` package.
func NewBundleFromTarGZ(path string) (*Bundle, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("error while opening %s: %w", path, err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("error while creating gzip reader for %s: %w", path, err)
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)
	sourceFS, err := copyTarIntoMemory(tarReader)
	if err != nil {
		return nil, err
	}

	bundle := &Bundle{
		Source: sourceFS,
	}
	if err := bundle.ReadSource(); err != nil {
		return nil, fmt.Errorf("reading bundle data from %q: %w", path, err)
	}

	return bundle, nil
}

// UpdateManifest updates the bundle manifest to reflect the bundle data source
func (b *Bundle) UpdateManifest() error {
	b.Manifest = &Manifest{
		Metadata: b.Metadata,
		Files:    b.Files,
	}
	return nil
}

// ReadSource loads the data from the mindpak source filesystem
func (b *Bundle) ReadSource() error {
	if b.Source == nil {
		return fmt.Errorf("unable to read source, mindpak filesystem not defined")
	}

	b.Manifest = &Manifest{
		Metadata: &Metadata{},
		Files: &Files{
			Profiles:  []*File{},
			RuleTypes: []*File{},
		},
	}

	b.Files = &Files{
		Profiles:  []*File{},
		RuleTypes: []*File{},
	}

	err := fs.WalkDir(b.Source, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("reading %q: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasPrefix(path, PathProfiles+"/") &&
			!strings.HasPrefix(path, PathRuleTypes+"/") &&
			!strings.HasPrefix(path, ManifestFileName) {
			return fmt.Errorf("found unexpected entry in mindpak source: %q", path)
		}

		f, err := b.Source.Open(path)
		if err != nil {
			return fmt.Errorf("opening %q", path)
		}
		defer f.Close()

		if path == ManifestFileName {
			man := &Manifest{}
			if err := man.Read(f); err != nil {
				return fmt.Errorf("parsing manifest: %w", err)
			}
			b.Manifest = man
		}

		h := sha256.New()

		if _, err := io.Copy(h, f); err != nil {
			return fmt.Errorf("hashing %q", path)
		}

		fentry := File{
			Name: d.Name(),
			Hashes: map[HashAlgorithm]string{
				SHA256: fmt.Sprintf("%x", h.Sum(nil)),
			},
		}

		switch {
		case strings.HasPrefix(path, PathProfiles):
			b.Files.Profiles = append(b.Files.Profiles, &fentry)
		case strings.HasPrefix(path, PathRuleTypes):
			b.Files.RuleTypes = append(b.Files.RuleTypes, &fentry)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("traversing bundle data source: %w", err)
	}
	return nil
}

// Verify checks the contents of the bundle against its manifest
func (_ *Bundle) Verify() error {
	// FIXME(puerco): Implement
	return nil
}

func copyTarIntoMemory(tarReader *tar.Reader) (fs.StatFS, error) {
	// create the memfs instance, and create the directories we need
	sourceFS := afero.NewIOFS(afero.NewMemMapFs())
	if err := sourceFS.MkdirAll("/"+PathProfiles, 0700); err != nil {
		return nil, fmt.Errorf("error creating directory in memfs: %w", err)
	}
	if err := sourceFS.MkdirAll("/"+PathRuleTypes, 0700); err != nil {
		return nil, fmt.Errorf("error creating directory in memfs: %w", err)
	}

	var memFile afero.File
	// used for error handling inside the loop
	defer func() {
		if memFile != nil {
			_ = memFile.Close()
		}
	}()

	// copy each file in the tar into the memfs
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error while iterating through tar: %w", err)
		}

		// assumption: we do not care about anything other than regular files
		// filter out relative paths to keep the static analysis tools happy
		if strings.Contains(header.Name, "..") || header.Typeflag != tar.TypeReg {
			continue
		}

		memFile, err = sourceFS.Create(header.Name)
		if err != nil {
			return nil, fmt.Errorf("error while creating memfs file: %w", err)
		}

		if _, err = io.Copy(memFile, tarReader); err != nil {
			return nil, fmt.Errorf("error while copying file into memfs: %w", err)
		}

		if err = memFile.Close(); err != nil {
			return nil, fmt.Errorf("error while closing memfs file: %w", err)
		}
	}

	return sourceFS, nil
}
