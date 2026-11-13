// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package util provides helper functions for the minder CLI.
package util

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/util/jsonyaml"
)

var (
	// PyRequestsVersionRegexp is a regexp to match a line in a requirements.txt file, including the package version
	// and the comparison operators
	PyRequestsVersionRegexp = regexp.MustCompile(`\s*(>=|<=|==|>|<|!=)\s*(\d+(\.\d+)*(\*)?)`)
	// PyRequestsNameRegexp is a regexp to match a line in a requirements.txt file, parsing out the package name
	PyRequestsNameRegexp = regexp.MustCompile(`\s*(>=|<=|==|>|<|!=)`)
)

// TestWriter is a helper struct for testing
type TestWriter struct {
	Output string
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.Output += string(p)
	return len(p), nil
}

func getProtoMarshalOptions() protojson.MarshalOptions {
	return protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}

}

// GetJsonFromProto given a proto message, formats into json
func GetJsonFromProto(msg proto.Message) (string, error) {
	m := getProtoMarshalOptions()
	out, err := m.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GetYamlFromProto given a proto message, formats into yaml
func GetYamlFromProto(msg proto.Message) (string, error) {
	// first converts into json using the marshal options
	m := getProtoMarshalOptions()
	out, err := m.Marshal(msg)
	if err != nil {
		return "", err
	}

	// from byte, we get the raw message so we can convert into yaml
	var rawMsg json.RawMessage
	err = json.Unmarshal(out, &rawMsg)
	if err != nil {
		return "", err
	}
	yamlResult, err := jsonyaml.ConvertJsonToYaml(rawMsg)
	if err != nil {
		return "", err
	}
	return yamlResult, nil
}

// GetBytesFromProto given a proto message, formats into bytes
func GetBytesFromProto(message proto.Message) ([]byte, error) {
	m := getProtoMarshalOptions()
	return m.Marshal(message)
}

// IsHTTPURL reports whether f uses the HTTP or HTTPS scheme.
func IsHTTPURL(f string) bool {
	parsed, err := url.Parse(f)
	if err != nil {
		return false
	}
	return parsed.Opaque == "" && parsed.Hostname() != "" &&
		(parsed.Scheme == "http" || parsed.Scheme == "https")
}

// FileArgExt returns the extension of a local file path or an HTTP(S) URL path.
func FileArgExt(f string) string {
	if IsHTTPURL(f) {
		parsed, err := url.Parse(f)
		if err == nil {
			return filepath.Ext(parsed.Path)
		}
	}
	return filepath.Ext(f)
}

// ExpandedFile is a struct to hold a file path and whether it was expanded
type ExpandedFile struct {
	Path     string
	Expanded bool
}

// ExpandFileArgs expands a list of file arguments into a list of files.
// When sysFS is nil, the local operating system filesystem is used.
// If the file list contains "-" or regular files, it will leave them as-is.
// If the file list contains directories, it will expand them into a list of files.
func ExpandFileArgs(sysFS fs.FS, files ...string) ([]ExpandedFile, error) {
	var expandedFiles []ExpandedFile
	for _, f := range files {
		if f == "-" || IsHTTPURL(f) {
			expandedFiles = append(expandedFiles, ExpandedFile{
				Path:     f,
				Expanded: false,
			})
			continue
		}

		files, err := expandFromFS(sysFS, f)
		if err != nil {
			return nil, err
		}
		expandedFiles = append(expandedFiles, files...)
	}

	return expandedFiles, nil
}

func expandFromFS(sysFS fs.FS, f string) ([]ExpandedFile, error) {
	// Because fs.FS expects paths to be clean, slash-separated, and under the root, we
	// need to switch between os and fs.FS conventions depending on sysFS.
	var fi fs.FileInfo
	var err error
	if sysFS != nil {
		// fs.FS expects all paths to be slash-separated and cleaned
		f = path.Clean(filepath.ToSlash(f))
		fi, err = fs.Stat(sysFS, f)
	} else {
		fi, err = os.Stat(f)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}

	if !fi.IsDir() {
		return []ExpandedFile{{
			Path:     f,
			Expanded: false,
		}}, nil
	}

	var results []ExpandedFile
	walkFunc := func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error walking path %s: %w", p, walkErr)
		}
		if d.IsDir() {
			return nil
		}
		results = append(results, ExpandedFile{
			Path:     p,
			Expanded: true,
		})
		return nil
	}
	if sysFS != nil {
		err = fs.WalkDir(sysFS, f, walkFunc)
	} else {
		err = filepath.Walk(f, func(p string, info os.FileInfo, walkErr error) error {
			return walkFunc(p, fs.FileInfoToDirEntry(info), walkErr)
		})
	}
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}
	return results, nil
}

// ViperLogLevelToZerologLevel converts a viper log level to a zerolog log level
func ViperLogLevelToZerologLevel(viperLogLevel string) zerolog.Level {
	switch viperLogLevel {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel // Default to info level if the mapping is not found
	}
}
