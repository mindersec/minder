// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package util provides helper functions for the minder CLI.
package util

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/rs/zerolog"
	_ "github.com/signalfx/splunk-otel-go/instrumentation/github.com/lib/pq/splunkpq" // nolint
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

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
func GetJsonFromProto(msg protoreflect.ProtoMessage) (string, error) {
	m := getProtoMarshalOptions()
	out, err := m.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GetYamlFromProto given a proto message, formats into yaml
func GetYamlFromProto(msg protoreflect.ProtoMessage) (string, error) {
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
func GetBytesFromProto(message protoreflect.ProtoMessage) ([]byte, error) {
	m := getProtoMarshalOptions()
	return m.Marshal(message)
}

// OpenFileArg opens a file argument and returns a descriptor, closer, and error
// If the file is "-", it will return whatever is passed in as dashOpen and a no-op closer
func OpenFileArg(f string, dashOpen io.Reader) (desc io.Reader, closer func(), err error) {
	if f == "-" {
		desc = dashOpen
		closer = func() {}
		return desc, closer, nil
	}

	f = filepath.Clean(f)
	ftemp, err := os.Open(f)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file: %w", err)
	}

	closer = func() {
		err := ftemp.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error closing file: %v\n", err)
		}
	}

	desc = ftemp
	return desc, closer, nil
}

// ExpandedFile is a struct to hold a file path and whether it was expanded
type ExpandedFile struct {
	Path     string
	Expanded bool
}

// ExpandFileArgs expands a list of file arguments into a list of files.
// If the file list contains "-" or regular files, it will leave them as-is.
// If the file list contains directories, it will expand them into a list of files.
func ExpandFileArgs(files ...string) ([]ExpandedFile, error) {
	var expandedFiles []ExpandedFile
	for _, f := range files {
		if f == "-" {
			expandedFiles = append(expandedFiles, ExpandedFile{
				Path:     f,
				Expanded: false,
			})
			continue
		}

		f = filepath.Clean(f)
		fi, err := os.Stat(f)
		if err != nil {
			return nil, fmt.Errorf("error getting file info: %w", err)
		}

		expanded := fi.IsDir()
		err = filepath.Walk(f, func(path string, info os.FileInfo, walkerr error) error {
			if walkerr != nil {
				return fmt.Errorf("error walking path %s: %w", path, walkerr)
			}

			if info.IsDir() {
				return nil
			}

			expandedFiles = append(expandedFiles, ExpandedFile{
				Path:     path,
				Expanded: expanded,
			})

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory: %w", err)
		}
	}

	return expandedFiles, nil
}

// Int32FromString converts a string to an int32
func Int32FromString(v string) (int32, error) {
	if v == "" {
		return 0, fmt.Errorf("cannot convert empty string to int")
	}

	// convert string to int
	asInt32, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("error converting string to int: %w", err)
	}
	if asInt32 > math.MaxInt32 || asInt32 < math.MinInt32 {
		return 0, fmt.Errorf("integer %d cannot fit into int32", asInt32)
	}
	// already validated overflow
	// nolint:gosec
	return int32(asInt32), nil
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
