// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package rego

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"

	engif "github.com/stacklok/minder/internal/engine/interfaces"
)

// MinderRegoLib contains the minder-specific functions for rego
var MinderRegoLib = []func(res *engif.Result) func(*rego.Rego){
	FileExists,
	FileLs,
	FileRead,
}

func instantiateRegoLib(res *engif.Result) []func(*rego.Rego) {
	var lib []func(*rego.Rego)
	for _, f := range MinderRegoLib {
		lib = append(lib, f(res))
	}
	return lib
}

// FileExists is a rego function that checks if a file exists
// in the filesystem being evaluated (which comes from the ingester).
// It takes one argument, the path to the file to check.
// It's exposed as `file.exists`.
func FileExists(res *engif.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.exists",
			Decl: types.NewFunction(types.Args(types.S), types.B),
		},
		func(bctx rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
			var path string
			if err := ast.As(op1.Value, &path); err != nil {
				return nil, err
			}

			if res.Fs == nil {
				return nil, fmt.Errorf("cannot check file existence without a filesystem")
			}

			fs := res.Fs

			cpath := filepath.Clean(path)
			finfo, err := fs.Stat(cpath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return ast.BooleanTerm(false), nil
				}
				return nil, err
			}

			if finfo.IsDir() {
				return ast.BooleanTerm(false), nil
			}

			return ast.BooleanTerm(true), nil
		},
	)
}

// FileRead is a rego function that reads a file from the filesystem
// being evaluated (which comes from the ingester). It takes one argument,
// the path to the file to read. It's exposed as `file.read`.
func FileRead(res *engif.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.read",
			Decl: types.NewFunction(types.Args(types.S), types.S),
		},
		func(bctx rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
			var path string
			if err := ast.As(op1.Value, &path); err != nil {
				return nil, err
			}

			if res.Fs == nil {
				return nil, fmt.Errorf("cannot read file without a filesystem")
			}

			fs := res.Fs

			cpath := filepath.Clean(path)
			f, err := fs.Open(cpath)
			if err != nil {
				return nil, err
			}

			defer f.Close()

			all, rerr := io.ReadAll(f)
			if rerr != nil {
				return nil, rerr
			}

			allstr := ast.String(all)
			return ast.NewTerm(allstr), nil
		},
	)
}

// FileLs is a rego function that lists the files in a directory
// in the filesystem being evaluated (which comes from the ingester).
// It takes one argument, the path to the directory to list. It's exposed
// as `file.ls`.
// If the file is a file, it returns the file itself.
// If the file is a directory, it returns the files in the directory.
// If the file is a symlink, it follows the symlink and returns the files
// in the target.
func FileLs(res *engif.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.ls",
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		func(bctx rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
			var path string
			if err := ast.As(op1.Value, &path); err != nil {
				return nil, err
			}

			if res.Fs == nil {
				return nil, fmt.Errorf("cannot walk file without a filesystem")
			}

			fs := res.Fs

			// Check file information and return a list of files
			// and directories
			finfo, err := fs.Lstat(path)
			if err != nil {
				return fileLsHandleError(err)
			}

			// If the file is a file return the file itself
			if finfo.Mode().IsRegular() {
				return fileLsHandleFile(path)
			}

			// If the file is a directory return the files in the directory
			if finfo.Mode().IsDir() {
				return fileLsHandleDir(path, fs)
			}

			// If the file is a symlink, follow it
			if finfo.Mode()&os.ModeSymlink != 0 {
				// Get the target of the symlink
				target, err := fs.Readlink(path)
				if err != nil {
					return nil, err
				}

				// Get the file information of the target
				// NOTE: This overwrites the previous finfo
				finfo, err = fs.Lstat(target)
				if err != nil {
					return fileLsHandleError(err)
				}

				// If the target is a file return the file itself
				if finfo.Mode().IsRegular() {
					return fileLsHandleFile(target)
				}

				// If the target is a directory return the files in the directory
				if finfo.Mode().IsDir() {
					return fileLsHandleDir(target, fs)
				}
			}

			return nil, fmt.Errorf("cannot handle file type %s", finfo.Mode())
		},
	)
}

func fileLsHandleError(err error) (*ast.Term, error) {
	// If the file does not exist return null
	if errors.Is(err, os.ErrNotExist) {
		return ast.NullTerm(), nil
	}
	return nil, err
}

func fileLsHandleFile(path string) (*ast.Term, error) {
	return ast.NewTerm(
		ast.NewArray(
			ast.NewTerm(ast.String(path)),
		),
	), nil
}

func fileLsHandleDir(path string, fs billy.Filesystem) (*ast.Term, error) {
	// Get the files in the directory
	paths, err := fs.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// Create a list of files
	var files []*ast.Term
	for _, p := range paths {
		fpath := filepath.Join(path, p.Name())
		files = append(files, ast.NewTerm(ast.String(fpath)))
	}

	return ast.NewTerm(
		ast.NewArray(files...)), nil
}
