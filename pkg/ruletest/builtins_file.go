// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"fmt"
	"io/fs"

	"go.starlark.net/starlark"
	"golang.org/x/tools/txtar"
)

// builtinReadFile reads a file relative to the current Starlark test file.
// Note: This builtin only supports text files containing valid UTF-8 strings.
// Bytestreams that aren't valid UTF-8 strings are not supported.
func (tr *testCaseRunner) builtinReadFile(
	_ *starlark.Thread,
	b *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}

	// fs.ReadFile enforces unrooted, valid paths (preventing typical directory traversal)
	data, err := fs.ReadFile(tr.fs, path)
	if err != nil {
		return nil, fmt.Errorf("read_file: %w", err)
	}

	return starlark.String(string(data)), nil
}

// builtinTxtar parses a txtar string and returns a Starlark dictionary.
// Note: This builtin only supports text files containing valid UTF-8 strings.
// Bytestreams that aren't valid UTF-8 strings are not supported.
func builtinTxtar(
	_ *starlark.Thread,
	b *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var content string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "content", &content); err != nil {
		return nil, err
	}

	archive := txtar.Parse([]byte(content))
	dict := starlark.NewDict(len(archive.Files))

	for _, f := range archive.Files {
		key := starlark.String(f.Name)
		val := starlark.String(string(f.Data))
		if err := dict.SetKey(key, val); err != nil {
			return nil, err
		}
	}

	return dict, nil
}
