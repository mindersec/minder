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

	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// Printer provides an interface for passing a printf-like function.
type Printer func(string, ...any)

// ResourcesFromPaths collects
func ResourcesFromPaths(printer Printer, paths ...string) ([]minderv1.ResourceMeta, error) {
	files, err := util.ExpandFileArgs(paths...)
	if err != nil {
		return nil, fmt.Errorf("error expanding args: %w", err)
	}

	objects := make([]minderv1.ResourceMeta, 0, len(files))
	for _, file := range files {
		resources, err := ReadFromPath(printer, file)
		if err != nil && !errors.Is(err, minderv1.ErrNotAResource) {
			return nil, err
		}
		objects = append(objects, resources...)

	}
	return objects, nil
}

// ReadFromPath reads a _single_ file (possibly expanded) and returns a set of Minder
// resources (often 1, but YAML files may include multiple documents) decoded by file
// type.  If "-" is passed as the filename, stdin will be read as a YAML document.
//
// TODO: do we want to return minderv1.ErrNotAResource or minderv1.ErrInvalidResource
// on parse erros?  Right now, we simply skip them (from ResourcesFromPaths' behavior).
func ReadFromPath(printer Printer, file util.ExpandedFile) ([]minderv1.ResourceMeta, error) {
	objects := []minderv1.ResourceMeta{}
	var input Decoder
	if file.Path == "-" {
		input = yaml.NewDecoder(os.Stdin)
	} else {
		var closer io.Closer
		input, closer = DecoderForFile(file.Path)
		if input == nil {
			// Not a valid file type, skip it.
			return objects, nil
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
	return objects, nil
}
