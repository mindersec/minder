//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tests runs the fga CLI tests against each file in this directory.
// It assumes the `fga` binary is in the PATH, e.g. from `make bootstrap`
package tests

import (
	"bytes"
	"path/filepath"
	"testing"

	// We need the init() function here, but we can't copy rootCmd because it is private
	_ "github.com/openfga/cli/cmd"
	"github.com/openfga/cli/cmd/model"
)

func TestFGA(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.tests.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test files found")
	}
	for _, f := range files {
		file := f
		t.Run(file, func(t *testing.T) {
			// We invoke cobra commands directly, which reference some global state in FGA.

			output, err := runFgaModelTest(file)
			if err != nil {
				t.Logf("%s failed: %s, output:\n%s", file, err, string(output))
			} else {
				t.Logf("%s succeeded, output:\n%s", file, string(output))
			}
		})
	}
}

// This is a little slimy, but we pull in the FGA CLI which implements "model test"
// directly here, so we don't need to bootstrap the external command.
//
// This means we're using the CLI's exported `ModelCmd` as an API, which is icky.
func runFgaModelTest(filename string) ([]byte, error) {
	cmd := *model.ModelCmd
	cmd.SetArgs([]string{"test", "--tests", filename})
	buffer := new(bytes.Buffer)
	cmd.SetOutput(buffer)

	if err := cmd.Execute(); err != nil {
		return buffer.Bytes(), err
	}
	return buffer.Bytes(), nil
}
