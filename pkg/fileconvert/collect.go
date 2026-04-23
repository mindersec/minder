// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fileconvert provides functions for marshalling Minder proto objects
// to and from on-disk formats like YAML.
package fileconvert

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-billy/v5"
	billyutil "github.com/go-git/go-billy/v5/util"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// Printer provides an interface for passing a printf-like function.
type Printer func(string, ...any)

// ResourcesFromFilesystem collects resources from a directory in a billy filesystem.
func ResourcesFromFilesystem[T proto.Message](printer Printer, fs billy.Filesystem, root string) ([]T, error) {
	if printer == nil {
		printer = func(string, ...any) {}
	}

	paths := make([]string, 0)
	err := billyutil.Walk(fs, root, func(path string, info iofs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml", ".json":
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", root, err)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	sort.Strings(paths)

	objects := make([]T, 0, len(paths))
	for _, path := range paths {
		resource, err := ReadResourceFromFile[T](fs, path)
		if err != nil {
			printer("Skipping invalid file %s: %v\n", path, err)
			continue
		}
		objects = append(objects, resource)
	}

	return objects, nil
}

// ResourcesFromPaths collects
func ResourcesFromPaths(printer Printer, paths ...string) ([]minderv1.ResourceMeta, error) {
	files, err := util.ExpandFileArgs(paths...)
	if err != nil {
		return nil, fmt.Errorf("error expanding args: %w", err)
	}

	objects := make([]minderv1.ResourceMeta, 0, len(files))
	for _, file := range files {
		var input Decoder
		if file.Path == "-" {
			input = yaml.NewDecoder(os.Stdin)
		} else {
			var closer io.Closer
			input, closer = DecoderForFile(file.Path)
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
