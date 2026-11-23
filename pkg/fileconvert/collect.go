// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fileconvert provides functions for marshalling Minder proto objects
// to and from on-disk formats like YAML.
package fileconvert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// Printer provides an interface for passing a printf-like function.
type Printer func(string, ...any)

// ResourcesFromPaths collects resources from paths or URLs in sysFS, canceling any remote
// file requests when ctx is canceled. When sysFS is nil, the local operating system
// filesystem is used.
func ResourcesFromPaths(ctx context.Context, sysFS fs.FS, printer Printer, paths ...string) ([]minderv1.ResourceMeta, error) {
	files, err := util.ExpandFileArgs(sysFS, paths...)
	if err != nil {
		return nil, fmt.Errorf("error expanding args: %w", err)
	}

	objects := make([]minderv1.ResourceMeta, 0, len(files))
	for _, file := range files {
		resources, err := ReadFromPath(ctx, sysFS, printer, file)
		if err != nil && !errors.Is(err, minderv1.ErrNotAResource) {
			return nil, err
		}
		objects = append(objects, resources...)

	}
	return objects, nil
}

// ReadFromPath reads one file and cancels remote requests when ctx is canceled.
// When sysFS is nil, the local operating system filesystem is used.
// If "-" is passed as the filename, stdin will be read as a YAML document.
// If the filename is an HTTP URL, it will be fetched over the network.
func ReadFromPath(ctx context.Context, sysFS fs.FS, printer Printer, file util.ExpandedFile) ([]minderv1.ResourceMeta, error) {
	objects := []minderv1.ResourceMeta{}
	var input Decoder
	if file.Path == "-" {
		input = yaml.NewDecoder(os.Stdin)
	} else if util.IsHTTPURL(file.Path) {
		builder := decoderBuilder(file.Path)
		if builder == nil {
			return nil, fmt.Errorf(
				"unsupported file extension %q for URL %q; supported extensions are .json, .yaml, .yml, and .rego",
				util.FileArgExt(file.Path), file.Path,
			)
		}
		reader, closer, err := util.OpenFileArgContext(ctx, file.Path, nil)
		if err != nil {
			return nil, fmt.Errorf("error opening file %s: %w", file.Path, err)
		}
		defer closer()
		input = builder(reader)
	} else {
		var closer io.Closer
		input, closer = DecoderForFile(sysFS, file.Path)
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
				if printer != nil {
					printer("Skipping expanded file %s due to error %s\n", file.Path, err)
				}
				break
			}
			return nil, fmt.Errorf("error reading resource from file %s: %w", file.Path, err)
		}
		objects = append(objects, resource)
	}
	return objects, nil
}
