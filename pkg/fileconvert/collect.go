// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fileconvert provides functions for marshalling Minder proto objects
// to and from on-disk formats like YAML.
package fileconvert

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/go-git/go-billy/v5"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// Printer provides an interface for passing a printf-like function.
type Printer func(string, ...any)

type filePath struct {
	Path     string
	Expanded bool
}

// ResourcesFromPaths collects
func ResourcesFromPaths(vfs billy.Filesystem, printer Printer, paths ...string) ([]minderv1.ResourceMeta, error) {
	if printer == nil {
		printer = func(string, ...any) {}
	}

	expandedFiles, err := util.ExpandFileArgs(vfs, paths...)
	if err != nil {
		return nil, fmt.Errorf("error expanding args: %w", err)
	}
	files := make([]filePath, 0, len(expandedFiles))
	for _, file := range expandedFiles {
		files = append(files, filePath{Path: file.Path, Expanded: file.Expanded})
	}

	objects := make([]minderv1.ResourceMeta, 0, len(files))
	for _, file := range files {
		var input Decoder
		var closer io.Closer
		if file.Path == "-" {
			if vfs != nil {
				return nil, fmt.Errorf("stdin is not supported with filesystem-backed reads")
			}
			input = yaml.NewDecoder(os.Stdin)
		} else {
			input, closer = decoderForPath(vfs, file.Path)
			if input == nil {
				// Not a valid file type, skip it.
				continue
			}
			defer closer.Close()
		}

		for i := 0; ; i = i + 1 {
			resource, err := ReadResource(input)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				if file.Expanded && i == 0 {
					// Skip files expanded from directories where the contents aren't valid
					printer("Skipping expanded file %s due to error %s\n", file.Path, err)
					break
				}
				return nil, fmt.Errorf("error reading resource from file %s: %w", file.Path, err)
			}
			objects = append(objects, resource)
		}
	}
	return objects, nil
}
