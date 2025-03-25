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
	"github.com/go-git/go-billy/v5/helper/iofs"
	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/types"
	"github.com/pelletier/go-toml/v2"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/stacklok/frizbee/pkg/replacer"
	"github.com/stacklok/frizbee/pkg/utils/config"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/deps/scalibr"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/flags"
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
	BaseFileArchive,
	ListGithubActions,
	ParseYaml,
	ParseToml,
	JQIsTrue,
}

// MinderRegoLibExperiments contains Minder-specific functions which
// should only be exposed when the given experiment is enabled.
var MinderRegoLibExperiments = map[flags.Experiment][]func(res *interfaces.Result) func(*rego.Rego){
	flags.GitPRDiffs: {
		BaseFileExists,
		BaseFileLs,
		BaseFileLsGlob,
		BaseFileHTTPType,
		BaseFileRead,
		BaseFileWalk,
		BaseListGithubActions,
	},
	flags.DependencyExtract: {
		DependencyExtract,
		BaseDependencyExtract,
	},
}

func instantiateRegoLib(ctx context.Context, featureFlags flags.Interface, res *interfaces.Result) []func(*rego.Rego) {
	var lib []func(*rego.Rego)
	for _, f := range MinderRegoLib {
		lib = append(lib, f(res))
	}
	for flag, funcs := range MinderRegoLibExperiments {
		if flags.Bool(ctx, featureFlags, flag) {
			for _, f := range funcs {
				lib = append(lib, f(res))
			}
		}
	}
	return lib
}

// FileExists adds the `file.exists` function to the Rego engine.
func FileExists(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.exists",
			Description: `file.exists checks if a file exists
			in the filesystem being evaluated (which comes from the ingester).
			It takes one argument, the path to the file to check, and returns
			a boolean.`,
			Decl: types.NewFunction(types.Args(types.S), types.B),
		},
		fsExists(res.Fs),
	)
}

// BaseFileExists adds the `base_file.exists` function to the Rego engine.
func BaseFileExists(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.exists",
			Description: `base_file.exists checks if a file exists
			in the pre-change filesystem being evaluated (in a pull_request or
			other diff context).
			It takes one argument, the path to the file to check, and returns
			a boolean.`,
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

// FileRead adds the `file.read` function to the Rego engine.
func FileRead(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.read",
			Description: `file.read reads a file from the filesystem
			being evaluated (which comes from the ingester).
			It takes one argument, the path to the file to read, and returns
			the file contents as a string.`,
			Decl: types.NewFunction(types.Args(types.S), types.S),
		},
		fsRead(res.Fs),
	)
}

// BaseFileRead adds the `base_file.read` function to the Rego engine.
func BaseFileRead(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.read",
			Description: `base_file.read reads a file from the pre-change
			filesystem in a pull_request or other diff context.
			It takes one argument, the path to the file to read, and returns
			the file contents as a string.`,
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

// FileLs adds the `file.ls` function to the Rego engine.
func FileLs(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.ls",
			Description: `file.ls lists the files in a directory in the
			filesystem being evaluated (which comes from the ingester).
		    It takes one argument, the path to the directory to list, and
			returns a list of strings.
			If the file is a file, it returns a one-element list with the filename.
			If the file is a directory, it returns the files in the directory.
			If the file is a symlink, it follows the symlink and returns the files
			in the target.`,
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
			Description: `base_file.ls lists the files in a directory in the
			pre-change filesystem being evaluated (in a pull_request or other diff context).
			It takes one argument, the path to the directory to list, and
			returns a list of strings.
			If the file is a file, it returns a one-element list with the filename.
			If the file is a directory, it returns the files in the directory.
			If the file is a symlink, it follows the symlink and returns the files
			in the target.`,
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

// FileLsGlob adds the `file.ls_glob` function to the Rego engine.
func FileLsGlob(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.ls_glob",
			Description: `file.ls_glob lists the files matching a glob in a
			directory in the filesystem being evaluated (which comes from the ingester).
			It takes one argument, the path to the pattern to match, and
			returns a list of strings for each file that matches the glob.`,
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsLsGlob(res.Fs),
	)
}

// BaseFileLsGlob adds the `base_file.ls_glob` function to the Rego engine.
func BaseFileLsGlob(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.ls_glob",
			Description: `file.ls_glob lists the files matching a glob in a
			directory in the pre-change filesystem being evaluated (in a pull_request
			or other diff context).
			It takes one argument, the path to the pattern to match, and
			returns a list of strings for each file that matches the glob.`,
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

// FileWalk adds the `file.walk` function to the Rego engine.
func FileWalk(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.walk",
			Description: `file.walk lists the files underneath a directory in
			the filesystem being evaluated (which comes from the ingester).
			It takes one argument, the base directory to walk, and returns
			a list of filenames as strings for each file that is found.`,
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsWalk(res.Fs),
	)
}

// BaseFileWalk adds the `base_file.walk` function to the Rego engine.
func BaseFileWalk(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.walk",
			Description: `base_file.walk lists the files underneath a directory
			in the pre-change filesystem being evaluated (in a pull_request or other
			diff context).
			It takes one argument, the base directory to walk, and returns
			a list of filenames as strings for each file that is found.`,
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

// FileArchive adds the 'file.archive` function to the Rego engine.
func FileArchive(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.archive",
			Description: `file.archive packages a set of files and directories
			from the filesystem being evaluated into a .tar.gz archive.
			It takes one argument: a list of file or directory paths to include,
			and returns the archive as a binary string.`,
			Decl: types.NewFunction(types.Args(types.NewArray(nil, types.S)), types.S),
		},
		fsArchive(res.Fs),
	)
}

// BaseFileArchive adds the 'base_file.archive` function to the Rego engine.
func BaseFileArchive(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.archive",
			Description: `base_file.archive packages a set of files and directories
			from the pre-change filesystem being evaluated (in a pull_request or
			other diff context) into a .tar.gz archive.
			It takes one argument: a list of file or directory paths to include,
			and outputs the archive as a binary string.`,
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

// ListGithubActions adds the `github_workflow.ls_actions` function to the Rego engine.
// The frizbee library guarantees that the actions are unique.
func ListGithubActions(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "github_workflow.ls_actions",
			Description: `github_workflow.ls_actions lists the GitHub Actions
			references in all files within a directory in the filesystem being
			evaluated (which comes from the ingester).
			It takes a single argument, the path to the directory to list, and
			returns a set of strings, each string being the name of an action.`,
			Decl: types.NewFunction(types.Args(types.S), types.NewSet(types.S)),
		},
		fsListGithubActions(res.Fs),
	)
}

// BaseListGithubActions adds the `github_workflow.base_ls_actions` function to the Rego engine.
// The frizbee library guarantees that the actions are unique.
func BaseListGithubActions(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "github_workflow.base_ls_actions",
			Description: `github_workflow.base_ls_actions lists the GitHub Actions
			references in all files within a directory in the pre-change filesystem being
			evaluated (in a pull_request or other diff context).
			It takes a single argument, the path to the directory to list, and
			returns a set of strings, each string being the name of an action.`,
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

// FileHTTPType adds the `file.http_type` function to the Rego engine.
func FileHTTPType(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.http_type",
			Description: `file.http_type determines the HTTP (MIME) type of
			a file in the filesystem being evaluated (which comes from the ingester).
			It takes one argument, the path to the file to check, and returns
			the MIME type as a string, defaulting to "application/octet-stream" if the
			type cannot be determined.`,
			Decl: types.NewFunction(types.Args(types.S), types.S),
		},
		fsHTTPType(res.Fs),
	)
}

// BaseFileHTTPType adds the `base_file.http_type` function to the Rego engine.
func BaseFileHTTPType(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.http_type",
			Description: `base_file.http_type determines the HTTP (MIME) type of
			a file in the filesystem being evaluated (in a pull_request or other diff context).
			It takes one argument, the path to the file to check, and returns
			the MIME type as a string, defaulting to "application/octet-stream" if the
			type cannot be determined.`,
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

// JQIsTrue adds the `jq.is_true` function to the Rego engine.
func JQIsTrue(_ *interfaces.Result) func(*rego.Rego) {
	return rego.Function2(
		&rego.Function{
			Name: "jq.is_true",
			Description: `jq.is_true runs a boolean jq query against supplied object data.
			It takes two arguments: the object data (such as parsed YAML), and
			the jq query as a string, and it returns a boolean indicating whether
			the query matches the object data.`,
			Decl: types.NewFunction(types.Args(types.A, types.S), types.B),
		},
		jqIsTrue,
	)
}

func jqIsTrue(_ rego.BuiltinContext, parsedYaml *ast.Term, query *ast.Term) (*ast.Term, error) {
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
}

// ParseYaml adds the `parse_yaml` function to the Rego engine.
func ParseYaml(_ *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "parse_yaml",
			Description: `parse_yaml parses a YAML string into object data.
			It takes one argument: the YAML content as a string, and returns the
			parsed YAML data as an object.`,
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		parseYaml,
	)
}

func parseYaml(_ rego.BuiltinContext, yamlContent *ast.Term) (*ast.Term, error) {
	var yamlStr string

	// Convert the YAML input from the term into a string
	if err := ast.As(yamlContent.Value, &yamlStr); err != nil {
		return nil, err
	}

	// Convert the YAML string into a Go interface
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
}

// ParseToml adds the `parse_toml` function to the Rego engine.
func ParseToml(_ *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "parse_toml",
			Description: `parse_toml parses a TOML string into object data.
			It takes one argument: the TOML content as a string, and returns the
			parsed TOML data as an object.`,
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		parseToml,
	)
}

func parseToml(_ rego.BuiltinContext, content *ast.Term) (*ast.Term, error) {
	var tomlStr string

	// Convert the TOML input from the term into a string
	if err := ast.As(content.Value, &tomlStr); err != nil {
		return nil, err
	}

	// Convert the TOML string into a Go interface
	var jsonObj any
	err := toml.Unmarshal([]byte(tomlStr), &jsonObj)
	if err != nil {
		return nil, fmt.Errorf("error converting YAML to JSON: %w", err)
	}

	// Convert the Go value to an ast.Value
	value, err := ast.InterfaceToValue(jsonObj)
	if err != nil {
		return nil, fmt.Errorf("error converting to AST value: %w", err)
	}

	return ast.NewTerm(value), nil
}

// DependencyExtract adds the `file.deps` function to the Rego engine.
func DependencyExtract(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "file.deps",
			Description: `file.deps extracts dependencies from a file or subtree
			of the filesystem being evaluated (which comes from the ingester).
			It takes one argument: the path to the file or subtree to be scanned,
			and returns the extracted dependencies in the form of a protobom SBOM
			with "nodes", but not "edges".  In particular, the SBOM Nodes will be
			stored as an array of objects in ".node_list.nodes" within the returned object.`,
			// TODO: The return type is types.A, but it should be types.NewObject(...)
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsExtractDeps(res.Fs),
	)
}

// BaseDependencyExtract adds the `base_file.deps` function to the Rego engine.
func BaseDependencyExtract(res *interfaces.Result) func(*rego.Rego) {
	return rego.Function1(
		&rego.Function{
			Name: "base_file.deps",
			Description: `base_file.deps extracts dependencies from a file or subtree
			of the filesystem being evaluated (in a pull_request or other diff context).
			It takes one argument: the path to the file or subtree to be scanned,
			and returns the extracted dependencies in the form of a protobom SBOM
			with "nodes", but not "edges".  In particular, the SBOM Nodes will be
			stored as an array of objects in ".node_list.nodes" within the returned object.`,
			// TODO: The return type is types.A, but it should be types.NewObject(...)
			Decl: types.NewFunction(types.Args(types.S), types.A),
		},
		fsExtractDeps(res.BaseFs),
	)
}

func fsExtractDeps(vfs billy.Filesystem) func(rego.BuiltinContext, *ast.Term) (*ast.Term, error) {
	return func(bctx rego.BuiltinContext, op1 *ast.Term) (*ast.Term, error) {
		var path string
		if err := ast.As(op1.Value, &path); err != nil {
			return nil, err
		}

		if vfs == nil {
			return nil, fmt.Errorf("file system is not available")
		}

		// verify the file or path exists
		target, err := vfs.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("file or path %q does not exist", path)
			}
			return nil, err
		}

		// vfs.Chroot() only works on directories, so if we have a file, copy
		// it to a new vfs.
		if !target.IsDir() {
			sourceFile, err := vfs.Open(path)
			if err != nil {
				return nil, fmt.Errorf("failed to open file %q", path)
			}
			defer sourceFile.Close()

			newVfs := memfs.New()
			basename := filepath.Base(path)
			file, err := newVfs.Create(basename)
			if err != nil {
				return nil, fmt.Errorf("failed to create file %q", basename)
			}
			defer file.Close()
			_, err = io.Copy(file, sourceFile)
			if err != nil {
				return nil, fmt.Errorf("failed to copy file %q", path)
			}
			vfs = newVfs
			path = ""
		}

		// construct a scalibr extractor
		extractor := scalibr.NewExtractor()

		if path != "" {
			vfs, err = vfs.Chroot(path)
			if err != nil {
				return nil, err
			}
		}

		res, err := extractor.ScanFilesystem(bctx.Context, iofs.New(vfs))
		if err != nil {
			return nil, fmt.Errorf("failed to scan filesystem: %v", err)
		}
		// put in an SBOM wrapper
		sbom := &sbom.Document{
			NodeList: res,
		}
		astValue, err := ast.InterfaceToValue(sbom)

		return &ast.Term{Value: astValue}, err
	}
}
