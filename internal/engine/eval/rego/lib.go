// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
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

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"

	engif "github.com/stacklok/mediator/internal/engine/interfaces"
)

var mediatorRegoLib = []func(res *engif.Result) func(*rego.Rego){
	FileExists,
	FileRead,
}

func instantiateRegoLib(res *engif.Result) []func(*rego.Rego) {
	var lib []func(*rego.Rego)
	for _, f := range mediatorRegoLib {
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
