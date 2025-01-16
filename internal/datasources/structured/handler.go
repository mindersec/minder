// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package structured

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/structpb"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

const (
	jsonType decoderType = "json"
	yamlType decoderType = "yaml"
	tomlType decoderType = "toml"
)

type decoder interface {
	Parse(io.Reader) (any, error)
	Extensions() []string
}

type decoderType string

// Catalog of decoders enabled by default
var decoders = map[decoderType]decoder{
	jsonType: &jsonDecoder{},
	yamlType: &yamlDecoder{},
	tomlType: &tomlDecoder{},
}

var _ v1datasources.DataSourceFuncDef = (*structHandler)(nil)

// ErrorNoFileMatchInPath triggers if the path specification can't match a
// file in the filesystem received by the data source.
var ErrorNoFileMatchInPath = errors.New("no file matched through path specification")

type structHandler struct {
	Path *minderv1.StructDataSource_Def_Path
}

func newHandlerFromDef(def *minderv1.StructDataSource_Def) (*structHandler, error) {
	if def == nil {
		return nil, errors.New("data source handler definition is nil")
	}

	return &structHandler{
		Path: def.GetPath(),
	}, nil
}

// openFirstAlternative tries to open the main path and return an open file. If
// not found, it will try the defined alternatives returning the first one.
// If paths are directories, they will be ignored. Returns an error if no path
// corresponds to a file that can be opened.
func openFirstAlternative(fs billy.Filesystem, mainPath string, alternatives []string) (billy.File, error) {
	if mainPath == "" && len(alternatives) == 0 {
		return nil, errors.New("no file specified in data source definition")
	}
	if mainPath != "" {
		alternatives = append([]string{mainPath}, alternatives...)
	}

	for _, p := range alternatives {
		s, err := fs.Stat(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("error checking path: %w", err)
		}

		if s.IsDir() {
			continue
		}

		f, err := fs.Open(p)
		if err != nil {
			return nil, fmt.Errorf("opening file: %w", err)
		}

		return f, nil
	}
	return nil, ErrorNoFileMatchInPath
}

// parseFileAlternatives takes a path and alternative locations and parses
// the first available
func parseFileAlternatives(fs billy.Filesystem, mainPath string, alternatives []string) (any, error) {
	f, err := openFirstAlternative(fs, mainPath, alternatives)
	if err != nil {
		// If no file was found, we don't return an error but nil
		// we want rules to not error but to get a blank struct
		if errors.Is(err, ErrorNoFileMatchInPath) {
			log.Info().Err(err).Msg("error validating datasource function arguments")
			return nil, nil
		}
		return nil, err
	}
	return parseFile(f)
}

// parseFile parses an open file using the configured parsers
func parseFile(f billy.File) (any, error) {
	// Get the file extension, perhaps we can shortcut before trying
	// to brute force through all decoders
	ext := filepath.Ext(f.Name())
	tried := map[decoderType]struct{}{}
	for t, d := range decoders {
		exts := d.Extensions()
		if slices.Contains(exts, strings.ToLower(ext)) {
			if _, err := f.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("unable to rewind file")
			}
			res, err := d.Parse(f)
			if err == nil {
				return res, nil
			}
			tried[t] = struct{}{}
		}
	}

	// no dice, try the rest of the decoders
	for t, d := range decoders {
		if _, ok := tried[t]; ok {
			continue
		}
		if _, err := f.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("unable to rewind file")
		}
		res, err := d.Parse(f)
		if err == nil {
			return res, nil
		}
	}
	return nil, errors.New("unable to parse structured data with any of the available decoders")
}

// Call parses the structured data from the billy filesystem in the context
func (sh *structHandler) Call(ctx context.Context, ingest *interfaces.Result, _ any) (any, error) {
	if ingest == nil || ingest.Fs == nil {
		return nil, fmt.Errorf("filesystem not found in execution context")
	}

	return parseFileAlternatives(ingest.Fs, sh.Path.GetFileName(), sh.Path.GetAlternatives())
}

func (*structHandler) GetArgsSchema() *structpb.Struct {
	return nil
}

// ValidateArgs is just a stub as the structured data source does not have arguments
func (_ *structHandler) ValidateArgs(any) error {
	return nil
}

// ValidateUpdate
func (_ *structHandler) ValidateUpdate(*structpb.Struct) error {
	return nil
}
