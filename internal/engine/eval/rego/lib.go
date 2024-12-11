// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
	billyutil "github.com/go-git/go-billy/v5/util"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"
	"github.com/stacklok/frizbee/pkg/replacer"
	"github.com/stacklok/frizbee/pkg/utils/config"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// MinderRegoLib contains the minder-specific functions for rego
var MinderRegoLib = []func(res *interfaces.Result) func(*rego.Rego){
	FileExists,
	FileLs,
	FileLsGlob,
	FileHTTPType,
	FileRead,
	FileWalk,
	FileArchive,
	ListGithubActions,
	BaseFileExists,
	BaseFileLs,
	BaseFileLsGlob,
	BaseFileHTTPType,
	BaseFileRead,
	BaseFileWalk,
	BaseListGithubActions,
	BaseFileArchive,
	ParseYaml,
	JQIsTrue,
}

func instantiateRegoLib(res *interfaces.Result) []func(*rego.Rego) {
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
func FileExists(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.exists",
			Decl: types.NewFunction(types.Args(types.S), types.B),
		},
		fsExists(res.Fs),
	)
}

// BaseFileExists is a rego function that checks if a file exists
// in the base filesystem from the ingester.  Base filesystems are
// typically associated with pull requests.
// It takes one argument, the path to the file to check.
// It's exposed as `base_file.exists`.
func BaseFileExists(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.exists",
			Decl: types.NewFunction(types.Args(types.S), types.B),
		},
		fsExists(res.BaseFs),
	)
}

func fsExists(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var path string
		if err := ast.As(op1.Value, &path); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("cannot check file existence without a filesystem")
		}

		cpath := filepath.Clean(path)
		_, err := vfs.Stat(cpath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return ast.BooleanTerm(false), nil
			}
			return nil, err
		}

		return ast.BooleanTerm(true), nil
	}
}

// FileRead is a rego function that reads a file from the filesystem
// being evaluated (which comes from the ingester). It takes one argument,
// the path to the file to read. It's exposed as `file.read`.
func FileRead(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.read",
			Decl: types.NewFunction(types.Args(types.S), types.S),
		},
		fsRead(res.Fs),
	)
}

// BaseFileRead is a rego function that reads a file from the
// base filesystem in a pull_request or other diff context.
// It takes one argument, the path to the file to read.
// It's exposed as `base_file.read`.
func BaseFileRead(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.read",
			Decl: types.NewFunction(types.Args(types.S), types.S),
		},
		fsRead(res.BaseFs),
	)
}

func fsRead(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var path string
		if err := ast.As(op1.Value, &path); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("cannot read file without a filesystem")
		}

		cpath := filepath.Clean(path)
		f, err := vfs.Open(cpath)
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
	}
}

// FileLs is a rego function that lists the files in a directory
// in the filesystem being evaluated (which comes from the ingester).
// It takes one argument, the path to the directory to list. It's exposed
// as `file.ls`.
// If the file is a file, it returns the file itself.
// If the file is a directory, it returns the files in the directory.
// If the file is a symlink, it follows the symlink and returns the files
// in the target.
func FileLs(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.ls",
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsLs(res.Fs),
	)
}

// BaseFileLs is a rego function that lists the files in a directory
// in the base filesystem being evaluated (in a pull_request or other
// diff context).  It takes one argument, the path to the directory to list.
// It's exposed as `base_file.ls`.
// If the file is a file, it returns the file itself.
// If the file is a directory, it returns the files in the directory.
// If the file is a symlink, it follows the symlink and returns the files
// in the target.
func BaseFileLs(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.ls",
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsLs(res.BaseFs),
	)
}

func fsLs(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var path string
		if err := ast.As(op1.Value, &path); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("cannot walk file without a filesystem")
		}

		// Check file information and return a list of files
		// and directories
		finfo, err := vfs.Lstat(path)
		if err != nil {
			return fileLsHandleError(err)
		}

		// If the file is a file return the file itself
		if finfo.Mode().IsRegular() {
			return fileLsHandleFile(path)
		}

		// If the file is a directory return the files in the directory
		if finfo.Mode().IsDir() {
			return fileLsHandleDir(path, vfs)
		}

		// If the file is a symlink, follow it
		if finfo.Mode()&os.ModeSymlink != 0 {
			// Get the target of the symlink
			target, err := vfs.Readlink(path)
			if err != nil {
				return nil, err
			}

			// Get the file information of the target
			// NOTE: This overwrites the previous finfo
			finfo, err = vfs.Lstat(target)
			if err != nil {
				return fileLsHandleError(err)
			}

			// If the target is a file return the file itself
			if finfo.Mode().IsRegular() {
				return fileLsHandleFile(target)
			}

			// If the target is a directory return the files in the directory
			if finfo.Mode().IsDir() {
				return fileLsHandleDir(target, vfs)
			}
		}

		return nil, fmt.Errorf("cannot handle file type %s", finfo.Mode())
	}
}

// FileLsGlob is a rego function that lists the files matching a glob in a directory
// in the filesystem being evaluated (which comes from the ingester).
// It takes one argument, the path to the pattern to match. It's exposed
// as `file.ls_glob`.
func FileLsGlob(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.ls_glob",
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsLsGlob(res.Fs),
	)
}

// BaseFileLsGlob is a rego function that lists the files matching a glob
// in a directory in the base filesystem being evaluated (in a pull_request
// or other diff context).
// It takes one argument, the path to the pattern to match. It's exposed
// as `base_file.ls_glob`.
func BaseFileLsGlob(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.ls_glob",
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsLsGlob(res.BaseFs),
	)
}

func fsLsGlob(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var path string
		if err := ast.As(op1.Value, &path); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("cannot walk file without a filesystem")
		}

		matches, err := billyutil.Glob(vfs, path)
		files := []*ast.Term{}

		for _, m := range matches {
			files = append(files, ast.NewTerm(ast.String(m)))
		}

		if err != nil {
			return nil, err
		}

		return ast.NewTerm(
			ast.NewArray(files...)), nil
	}
}

// FileWalk is a rego function that walks the files in a directory
// in the filesystem being evaluated (which comes from the ingester).
// It takes one argument, the path to the directory to walk. It's exposed
// as `file.walk`.
func FileWalk(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.walk",
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsWalk(res.Fs),
	)
}

// BaseFileWalk is a rego function that walks the files in a directory
// in the base filesystem being evaluated (in a pull_request or other
// diff context).
// It takes one argument, the path to the directory to walk. It's exposed
// as `base_file.walk`.
func BaseFileWalk(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.walk",
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsWalk(res.BaseFs),
	)
}

func fsWalk(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var path string
		if err := ast.As(op1.Value, &path); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("cannot walk file without a filesystem")
		}

		// if the path is a file, return the file itself
		// Check file information and return a list of files
		// and directories
		finfo, err := vfs.Lstat(path)
		if err != nil {
			return fileLsHandleError(err)
		}

		// If the file is a file return the file itself
		if finfo.Mode().IsRegular() {
			return fileLsHandleFile(path)
		}

		files := []*ast.Term{}
		err = billyutil.Walk(vfs, path, func(path string, info fs.FileInfo, err error) error {
			// skip if error
			if err != nil {
				return nil
			}

			// skip if directory
			if info.IsDir() {
				return nil
			}

			files = append(files, ast.NewTerm(ast.String(path)))
			return nil
		})
		if err != nil {
			return nil, err
		}

		return ast.NewTerm(
			ast.NewArray(files...)), nil
	}
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

func fileLsHandleDir(path string, bfs billy.Filesystem) (*ast.Term, error) {
	paths, err := bfs.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []*ast.Term
	for _, p := range paths {
		fpath := filepath.Join(path, p.Name())
		files = append(files, ast.NewTerm(ast.String(fpath)))
	}

	return ast.NewTerm(
		ast.NewArray(files...)), nil
}

// FileArchive packages a set of files form the the specified directory into
// a tarball.  It takes one argument: a list of file or directory paths to
// include, and outputs the tarball as a string.
// It's exposed as 'file.archive`.
func FileArchive(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.archive",
			Decl: types.NewFunction(types.Args(types.NewArray(nil, types.S)), types.S),
		},
		fsArchive(res.Fs),
	)
}

// BaseFileArchive packages a set of files form the the specified directory
// in the base filesystem (from a pull_request or other diff context) into
// a tarball.  It takes one argument: a list of file or directory paths to
// include, and outputs the tarball as a string.
// It's exposed as 'base_file.archive`.
func BaseFileArchive(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.archive",
			Decl: types.NewFunction(types.Args(types.NewArray(nil, types.S)), types.S),
		},
		fsArchive(res.BaseFs),
	)
}

func fsArchive(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var paths []string
		if err := ast.As(op1.Value, &paths); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("cannot archive files without a filesystem")
		}

		out := bytes.Buffer{}
		gzWriter := gzip.NewWriter(&out)
		defer gzWriter.Close()
		tarWriter := tar.NewWriter(gzWriter)
		defer tarWriter.Close()

		for _, f := range paths {
			fmt.Printf("++ Adding %s\n", f)
			err := billyutil.Walk(vfs, f, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				fileHeader, err := tar.FileInfoHeader(info, "")
				if err != nil {
					return err
				}
				fileHeader.Name = path
				// memfs doesn't return change times anyway, so zero them for consistency
				fileHeader.ModTime = time.Time{}
				fileHeader.AccessTime = time.Time{}
				fileHeader.ChangeTime = time.Time{}
				if err := tarWriter.WriteHeader(fileHeader); err != nil {
					return err
				}
				if info.Mode().IsRegular() {
					file, err := vfs.Open(path)
					if err != nil {
						return err
					}
					defer file.Close()
					if _, err := io.Copy(tarWriter, file); err != nil {
						return err
					}
				}
				fmt.Printf("  -> Added %s (%s): %d\n", path, info.Mode(), out.Len())
				return nil
			})
			if err != nil {
				return nil, err
			}
		}

		if err := tarWriter.Close(); err != nil {
			return nil, err
		}
		if err := gzWriter.Close(); err != nil {
			return nil, err
		}

		return ast.StringTerm(out.String()), nil
	}
}

// ListGithubActions is a rego function that lists the actions in a directory
// in the filesystem being evaluated (which comes from the ingester).
// It takes one argument, the path to the directory to list. It's exposed
// as `github_workflow.ls_actions`.
// The function returns a set of strings, each string being the name of an action.
// The frizbee library guarantees that the actions are unique.
func ListGithubActions(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "github_workflow.ls_actions",
			Decl: types.NewFunction(types.Args(types.S), types.NewSet(types.S)),
		},
		fsListGithubActions(res.Fs),
	)
}

// BaseListGithubActions is a rego function that lists the actions in a directory
// in the base filesystem being evaluated (in a pull_request or diff context).
// It takes one argument, the path to the directory to list. It's exposed
// as `github_workflow.base_ls_actions`.
// The function returns a set of strings, each string being the name of an action.
// The frizbee library guarantees that the actions are unique.
func BaseListGithubActions(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "github_workflow.base_ls_actions",
			Decl: types.NewFunction(types.Args(types.S), types.NewSet(types.S)),
		},
		fsListGithubActions(res.BaseFs),
	)
}

func fsListGithubActions(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var base string
		if err := ast.As(op1.Value, &base); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("cannot list actions without a filesystem")
		}

		var terms []*ast.Term

		// Parse the ingested file system and extract all action references
		r := replacer.NewGitHubActionsReplacer(&config.Config{})
		actions, err := r.ListPathInFS(vfs, base)
		if err != nil {
			return nil, err
		}

		// Save the action names
		for _, a := range actions.Entities {
			terms = append(terms, ast.StringTerm(a.Name))
		}

		return ast.SetTerm(terms...), nil
	}
}

// FileHTTPType is a rego function that returns the HTTP type of a file
// in the filesystem being evaluated (which comes from the ingester).
// It takes one argument, the path to the file to check. It's exposed
// as `file.http_type`.
func FileHTTPType(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.http_type",
			Decl: types.NewFunction(types.Args(types.S), types.S),
		},
		fsHTTPType(res.Fs),
	)
}

// BaseFileHTTPType is a rego function that returns the HTTP type of a file
// in the filesystem being evaluated (which comes from the ingester).
// It takes one argument, the path to the file to check. It's exposed
// as `base_file.http_type`.
func BaseFileHTTPType(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.http_type",
			Decl: types.NewFunction(types.Args(types.S), types.S),
		},
		fsHTTPType(res.BaseFs),
	)
}

func fsHTTPType(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var path string
		if err := ast.As(op1.Value, &path); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("cannot list actions without a filesystem")
		}

		cpath := filepath.Clean(path)
		f, err := vfs.Open(cpath)
		if err != nil {
			return nil, err
		}

		defer f.Close()

		buffer := make([]byte, 512)
		n, err := f.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}

		httpTyp := http.DetectContentType(buffer[:n])
		astHTTPTyp := ast.String(httpTyp)
		return ast.NewTerm(astHTTPTyp), nil
	}
}

// JQIsTrue is a rego function that accepts parsed YAML data and runs a jq query on it.
// The query is a string in jq format that returns a boolean.
// It returns a boolean indicating whether the jq query matches the parsed YAML data.
// It takes two arguments: the parsed YAML data as an AST term, and the jq query as a string.
// It's exposed as `jq.is_true`.
func JQIsTrue(_ *interfaces.Result) func(*rego.Rego) {
	return rego.Function2(
		&rego.Function{
			Name: "jq.is_true",
			// The function takes two arguments: parsed YAML data and the jq query string
			Decl: types.NewFunction(types.Args(types.A, types.S), types.B),
		},
		func(_ rego.BuiltinContext, parsedYaml *ast.Term, query *ast.Term) (*ast.Term, error) {
			var jqQuery string
			if err := ast.As(query.Value, &jqQuery); err != nil {
				return nil, err
			}

			// Convert the AST value back to a Go interface{}
			jsonObj, err := ast.JSON(parsedYaml.Value)
			if err != nil {
				return nil, fmt.Errorf("error converting AST to JSON: %w", err)
			}

			doesMatch, err := util.JQEvalBoolExpression(context.TODO(), jqQuery, jsonObj)
			if err != nil {
				return nil, fmt.Errorf("error running jq query: %w", err)
			}

			return ast.BooleanTerm(doesMatch), nil
		},
	)
}

// ParseYaml is a rego function that parses a YAML string into a structured data format.
// It takes one argument: the YAML content as a string.
// It returns the parsed YAML data as an AST term.
// It's exposed as `parse_yaml`.
func ParseYaml(_ *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "parse_yaml",
			// Takes one string argument (the YAML content) and returns any type
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		func(_ rego.BuiltinContext, yamlContent *ast.Term) (*ast.Term, error) {
			var yamlStr string

			// Convert the YAML input from the term into a string
			if err := ast.As(yamlContent.Value, &yamlStr); err != nil {
				return nil, err
			}

			// Convert the YAML string into a Go map
			var jsonObj any
			err := yaml.Unmarshal([]byte(yamlStr), &jsonObj)
			if err != nil {
				return nil, fmt.Errorf("error converting YAML to JSON: %w", err)
			}

			// Convert the Go value to an ast.Value
			value, err := ast.InterfaceToValue(jsonObj)
			if err != nil {
				return nil, fmt.Errorf("error converting to AST value: %w", err)
			}

			return ast.NewTerm(value), nil
		},
	)
}
